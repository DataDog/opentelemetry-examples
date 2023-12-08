package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/Shopify/sarama"
	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Consumer struct{}

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
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return tp, nil
}

var (
	logger, _ = NewZapLogger()
	tracer    trace.Tracer
)

const (
	kafkaServersEnv    = "KAFKA_SERVERS"
	groupIdEnv         = "GROUP_ID"
	autoOffsetResetEnv = "AUTO_OFFSET_RESET"
	topicEnv           = "KAFKA_TOPIC"
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
	groupId         = getEnv(groupIdEnv, "words-consumer")
	autoOffsetReset = getEnv(autoOffsetResetEnv, "smallest")
	kafkaTopic      = getEnv(topicEnv, "words")
)

func main() {
	if err := realMain(); err != nil {
		logger.Error("starting", zap.Error(err))
	}
}

func realMain() error {
	logger.Info(
		"config",
		zap.String("kafkaServer", kafkaServers),
		zap.String("topic", kafkaTopic),
	)
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
	tracer = tp.Tracer("words-consumer")

	if err := startConsumerGroup(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	logger.Info("Done consuming")
	return nil
}

func (consumer *Consumer) processMessage(
	ctx context.Context,
	messages []*sarama.ConsumerMessage,
	links []trace.Link,
) error {
	logger.Info("received messages", zap.Int("num", len(messages)))
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(800*time.Millisecond))
	defer cancel()

	ctx, span := tracer.Start(ctx, "counts", trace.WithLinks(links...))
	defer span.End()
	counts := make(map[string]int)
	for _, m := range messages {
		if _, ok := counts[string(m.Value)]; ok {
			counts[string(m.Value)]++
		}
		counts[string(m.Value)] = 1

	}
	logger.Info("counts", zap.String("counts", fmt.Sprintf("%v", counts)))

	return nil
}

func startConsumerGroup(ctx context.Context) error {
	consumerGroupHandler := Consumer{}
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
func (consumer *Consumer) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	var links []trace.Link
	var messages []*sarama.ConsumerMessage

	for message := range claim.Messages() {
		ctx := propagator.Extract(context.Background(), &recordCarrier{message.Headers})
		links = append(links, trace.LinkFromContext(ctx))
		messages = append(messages, message)
		if len(links) == 5 {
			err := consumer.processMessage(context.Background(), messages, links)
			if err != nil {
				return err
			}
			links = make([]trace.Link, 0)
			messages = make([]*sarama.ConsumerMessage, 0)

		}
	}

	return nil
}
