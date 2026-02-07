package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	// NOTE: otelsarama instrumentation requires github.com/Shopify/sarama.
	// The library has moved to github.com/IBM/sarama, but the OTel instrumentation
	// has not been updated yet. See: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4091
	"github.com/Shopify/sarama"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Consumer struct {
	redis *redis.Client
}

func NewZapLogger() (*zap.Logger, error) {
	loggingConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "json",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := loggingConfig.Build()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// propagator is set globally during InitTracerProvider.
// It is used for manual context extraction from Kafka message headers.
var propagator = propagation.TraceContext{}

// recordCarrier implements propagation.TextMapCarrier for Sarama record headers,
// enabling W3C Trace Context propagation across Kafka messages.
type recordCarrier struct {
	headers []*sarama.RecordHeader
}

func (r *recordCarrier) Get(key string) string {
	for _, h := range r.headers {
		if key == string(h.Key) {
			return string(h.Value)
		}
	}
	return ""
}

// Set stores the key-value pair.
func (r *recordCarrier) Set(key string, value string) {
	r.headers = append(r.headers, &sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	})
}

// Keys lists the keys stored in this carrier.
func (r *recordCarrier) Keys() []string {
	keys := make([]string, len(r.headers))
	for i, h := range r.headers {
		keys[i] = string(h.Key)
	}
	return keys
}

// InitTracerProvider creates and registers globally a new TracerProvider.
// It configures OTLP gRPC export and sets up W3C Trace Context + Baggage propagation.
// The OTEL_EXPORTER_OTLP_ENDPOINT env var controls the export destination.
func InitTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// The OTLP exporter reads OTEL_EXPORTER_OTLP_ENDPOINT from the environment.
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	// Build resource with service metadata per OTel semantic conventions.
	// OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES env vars are also read automatically.
	res, err := sdkresource.Merge(
		sdkresource.Default(),
		sdkresource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("calendar-consumer-go-otel"),
			attribute.String("messaging.system", "kafka"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	// Use both W3C TraceContext and Baggage propagators for distributed context propagation.
	// This ensures trace context is propagated correctly across Kafka messages
	// and is compatible with the Datadog Agent OTLP intake.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

var logger, _ = NewZapLogger()
var tracer trace.Tracer

const (
	kafkaServersEnv    = "KAFKA_SERVERS"
	groupIdEnv         = "GROUP_ID"
	autoOffsetResetEnv = "AUTO_OFFSET_RESET"
	topicEnv           = "KAFKA_TOPIC"
	redisHostEnv       = "REDIS_HOST"
	redisPortEnv       = "REDIS_PORT"
)

func getEnv(k string, df string) string {
	v := os.Getenv(k)
	if v == "" {
		return df
	}
	return v
}

var (
	kafkaServers    = getEnv(kafkaServersEnv, "localhost:9092")
	groupId         = getEnv(groupIdEnv, "calendar-consumer")
	autoOffsetReset = getEnv(autoOffsetResetEnv, "smallest")
	kafkaTopic      = getEnv(topicEnv, "calendar")
	redisHost       = getEnv(redisHostEnv, "localhost")
	redisPort       = getEnv(redisPortEnv, "6379")
	redisUrl        = fmt.Sprintf("%s:%s", redisHost, redisPort)
)

func main() {
	if err := realMain(); err != nil {
		logger.Error("starting", zap.Error(err))
	}
}

func realMain() error {
	logger.Info("config", zap.String("redisUrl", redisUrl), zap.String("kafkaServer", kafkaServers), zap.String("topic", kafkaTopic))

	// Handle both SIGINT and SIGTERM for graceful shutdown in containers.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	tp, err := InitTracerProvider(ctx)
	if err != nil {
		return err
	}

	// Graceful shutdown: flush remaining spans before exit.
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}()

	tracer = tp.Tracer("calendar-consumer")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: "",
		DB:       0,
		// Production: configure connection pool settings
		PoolSize:     10,
		MinIdleConns: 5,
	})
	defer rdb.Close()

	// Enable tracing instrumentation for Redis.
	if err := redisotel.InstrumentTracing(rdb, redisotel.WithTracerProvider(tp)); err != nil {
		return fmt.Errorf("instrumenting redis tracing: %w", err)
	}

	// Enable metrics instrumentation for Redis.
	if err := redisotel.InstrumentMetrics(rdb); err != nil {
		return fmt.Errorf("instrumenting redis metrics: %w", err)
	}

	if err := startConsumerGroup(ctx, rdb); err != nil {
		return err
	}

	<-ctx.Done()
	logger.Info("Done consuming")
	return nil
}

func (consumer *Consumer) processMessage(ctx context.Context, m *sarama.ConsumerMessage) error {
	logger.Info("received", zap.String("message", string(m.Value)),
		zap.String("topic", m.Topic),
		zap.Int32("partition", m.Partition),
		zap.Int64("offset", m.Offset))

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(800*time.Millisecond))
	defer cancel()

	// OTel messaging semantic convention: span name should be "<topic> process"
	// per https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/
	ctx, span := tracer.Start(ctx, fmt.Sprintf("%s process", m.Topic))
	defer span.End()

	dayOffset := rand.Intn(365)
	startDate := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local)
	day := startDate.AddDate(0, 0, dayOffset)

	d := day.Format(time.DateOnly)
	logger.Info("random date", zap.String("date", d))
	err := consumer.redis.Set(ctx, string(m.Value), d, 0).Err()
	if err != nil {
		logger.Error("unable to set key in redis", zap.Error(err))
		return fmt.Errorf("redis SET failed: %w", err)
	}
	return nil
}

func startConsumerGroup(ctx context.Context, rdb *redis.Client) error {
	consumerGroupHandler := Consumer{
		redis: rdb,
	}
	// OTel: Wrap the consumer group handler with otelsarama instrumentation.
	// This automatically creates spans for each consumed message and extracts
	// trace context from Kafka message headers.
	handler := otelsarama.WrapConsumerGroupHandler(&consumerGroupHandler)

	config := sarama.NewConfig()
	config.Version = sarama.V2_5_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroup([]string{kafkaServers}, groupId, config)
	if err != nil {
		return fmt.Errorf("starting consumer group: %w", err)
	}

	err = consumerGroup.Consume(ctx, []string{kafkaTopic}, handler)
	if err != nil {
		return fmt.Errorf("consuming via handler: %w", err)
	}
	return nil
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited.
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		// Extract trace context propagated via W3C TraceContext headers in Kafka messages.
		ctx := propagator.Extract(context.Background(), &recordCarrier{message.Headers})
		if err := consumer.processMessage(ctx, message); err != nil {
			logger.Error("Error processing message", zap.Error(err))
		}
		session.MarkMessage(message, "")
	}

	return nil
}
