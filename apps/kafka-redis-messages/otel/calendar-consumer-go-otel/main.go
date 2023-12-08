package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/Shopify/sarama"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

var propagator = propagation.TraceContext{}

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

	i := 0
	for k := range r.headers {
		keys[i] = string(k)
		i++
	}
	return keys
}

// InitTracer creates and registers globally a new TracerProvider.
func InitTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		logger.Fatal("Constructing new exporter", zap.Error(err))
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	tp, err := InitTracerProvider(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}()
	tracer = tp.Tracer("calendar-consumer")
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Enable tracing instrumentation.
	if err := redisotel.InstrumentTracing(rdb, redisotel.WithTracerProvider(tp)); err != nil {
		return err
	}

	// Enable metrics instrumentation.
	if err := redisotel.InstrumentMetrics(rdb); err != nil {
		return err
	}

	if err := startConsumerGroup(ctx, rdb); err != nil {
		return err
	}

	<-ctx.Done()
	logger.Info("Done consuming")
	return nil
}

func (consumer *Consumer) processMessage(ctx context.Context, m *sarama.ConsumerMessage) error {
	logger.Info("received", zap.String("message", string(m.Value)))
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(800*time.Millisecond))
	defer cancel()

	ctx, span := tracer.Start(ctx, "processMessage")
	defer span.End()

	dayOffset := rand.Intn(365)
	startDate := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local)
	day := startDate.AddDate(0, 0, dayOffset)

	d := day.Format(time.DateOnly)
	logger.Info("random date", zap.String("date", d))
	err := consumer.redis.Set(ctx, string(m.Value), d, 0).Err()
	if err != nil {
		logger.Error("unable to set key in redis", zap.Error(err))
		return err
	}
	return nil
}

func startConsumerGroup(ctx context.Context, rdb *redis.Client) error {
	consumerGroupHandler := Consumer{
		redis: rdb,
	}
	// Wrap instrumentation
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
		ctx := propagator.Extract(context.Background(), &recordCarrier{message.Headers})
		consumer.processMessage(ctx, message)
		session.MarkMessage(message, "")
	}

	return nil
}
