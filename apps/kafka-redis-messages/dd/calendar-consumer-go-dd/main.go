package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	// DD-specific: Sarama instrumentation via dd-trace-go wraps the Kafka consumer
	// to automatically create spans for each consumed message.
	saramatrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/IBM/sarama.v1"

	// DD-specific: Redis instrumentation via dd-trace-go wraps the Redis client
	// to automatically trace all Redis commands.
	redistrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/redis/go-redis.v9"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Server struct {
	redis redis.UniversalClient
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

var logger, _ = NewZapLogger()

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

	// DD-specific: Handle both SIGINT and SIGTERM for graceful shutdown in containers.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// DD-specific: Start the Datadog tracer. The tracer reads DD_AGENT_HOST,
	// DD_SERVICE, DD_ENV, and DD_VERSION from environment variables.
	tracer.Start()
	defer tracer.Stop()

	// DD-specific: Use redistrace.NewClient to automatically instrument Redis commands.
	// This creates spans for every Redis operation, tagged with the service name "redis".
	rdb := redistrace.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: "",
		DB:       0,
		// Production: configure connection pool settings
		PoolSize:     10,
		MinIdleConns: 5,
	},
		redistrace.WithServiceName("redis"),
	)
	defer rdb.Close()

	server := &Server{
		redis: rdb,
	}

	consumer, err := sarama.NewConsumer([]string{kafkaServers}, nil)
	if err != nil {
		return fmt.Errorf("creating kafka consumer: %w", err)
	}
	defer consumer.Close()

	// DD-specific: WrapConsumer instruments the Sarama consumer so that each
	// consumed message gets a span with Kafka metadata (topic, partition, offset).
	consumer = saramatrace.WrapConsumer(consumer)

	partitionConsumer, err := consumer.ConsumePartition(kafkaTopic, 0, sarama.OffsetNewest)
	if err != nil {
		return fmt.Errorf("creating partition consumer: %w", err)
	}

	logger.Info("Consumer started, waiting for messages...")

	// Start consuming messages with graceful shutdown
	for {
		select {
		case message := <-partitionConsumer.Messages():
			// DD-specific: Extract the trace context propagated via Kafka headers.
			// This links the consumer span to the producer span, creating a
			// distributed trace across Kafka.
			if spanctx, err := tracer.Extract(saramatrace.NewConsumerMessageCarrier(message)); err == nil {
				span, childCtx := tracer.StartSpanFromContext(ctx, "process-message", tracer.ChildOf(spanctx))
				err := server.processMessage(childCtx, message)
				span.Finish(tracer.WithError(err))
			} else {
				logger.Warn("Failed to extract trace context from message", zap.Error(err))
				// Process message without parent span context
				span, childCtx := tracer.StartSpanFromContext(ctx, "process-message")
				processErr := server.processMessage(childCtx, message)
				span.Finish(tracer.WithError(processErr))
			}
		case err := <-partitionConsumer.Errors():
			logger.Error("Error consuming message", zap.Error(err))
		case <-ctx.Done():
			logger.Info("Shutdown signal received, stopping consumer...")
			if err := partitionConsumer.Close(); err != nil {
				logger.Error("Error closing partition consumer", zap.Error(err))
			}
			logger.Info("Consumer stopped gracefully")
			return nil
		}
	}
}

func (s *Server) processMessage(ctx context.Context, m *sarama.ConsumerMessage) error {
	logger.Info("received", zap.String("message", string(m.Value)),
		zap.String("topic", m.Topic),
		zap.Int32("partition", m.Partition),
		zap.Int64("offset", m.Offset))

	dayOffset := rand.Intn(365)
	startDate := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local)
	day := startDate.AddDate(0, 0, dayOffset)

	d := day.Format(time.DateOnly)
	logger.Info("random date", zap.String("date", d))
	err := s.redis.Set(ctx, string(m.Value), d, 0).Err()
	if err != nil {
		logger.Error("unable to set key in redis", zap.Error(err))
		return fmt.Errorf("redis SET failed: %w", err)
	}
	return nil
}
