package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/Shopify/sarama"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	saramatrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/Shopify/sarama"

	redistrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-redis/redis.v8"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	_ "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	tracer.Start()
	defer tracer.Stop()

	rdb := redistrace.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: "", // no password set
		DB:       0,  // use default DB
	},
		redistrace.WithServiceName("redis"),
	)

	server := &Server{
		redis: rdb,
	}
	consumer, err := sarama.NewConsumer([]string{kafkaServers}, nil)
	if err != nil {
		return err
	}
	defer consumer.Close()

	consumer = saramatrace.WrapConsumer(consumer)

	partitionConsumer, err := consumer.ConsumePartition(kafkaTopic, 0, sarama.OffsetNewest)
	if err != nil {
		return err
	}
	// Start consuming messages
	for {
		select {
		case message := <-partitionConsumer.Messages():
			if spanctx, err := tracer.Extract(saramatrace.NewConsumerMessageCarrier(message)); err == nil {
				span, childCtx := tracer.StartSpanFromContext(ctx, "process-message", tracer.ChildOf(spanctx))
				err := server.processMessage(childCtx, message)
				span.Finish(tracer.WithError(err))
			}
		case err := <-partitionConsumer.Errors():
			logger.Error("Error consuming message:", zap.Error(err))
		case <-ctx.Done():
			logger.Info("Interrupt signal received, stopping consumer...")
			err := partitionConsumer.Close()
			if err != nil {
				return err
			}
			err = consumer.Close()
			if err != nil {
				return err
			}
			return nil
		}
	}

}

func (s *Server) processMessage(ctx context.Context, m *sarama.ConsumerMessage) error {
	logger.Info("received", zap.String("message", string(m.Value)))

	dayOffset := rand.Intn(365)
	startDate := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local)
	day := startDate.AddDate(0, 0, dayOffset)

	d := day.Format(time.DateOnly)
	logger.Info("random date", zap.String("date", d))
	err := s.redis.Set(ctx, string(m.Value), d, 0).Err()
	if err != nil {
		logger.Error("unable to set key in redis", zap.Error(err))
		return err
	}
	return nil
}
