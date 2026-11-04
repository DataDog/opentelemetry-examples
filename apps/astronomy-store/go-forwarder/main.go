package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}

func processingDelay() time.Duration {
	millis := min(max(500+150*rand.NormFloat64(), 0), 990)
	return time.Duration(millis * float64(time.Millisecond))
}

func waitForTopic(ctx context.Context, logger *slog.Logger, broker, topic string) error {
	for {
		conn, err := kafka.DialContext(ctx, "tcp", broker)
		if err == nil {
			partitions, readErr := conn.ReadPartitions(topic)
			conn.Close()
			if readErr == nil && len(partitions) > 0 {
				return nil
			}
		}
		logger.Info("waiting for topic to exist", slog.String("input_topic", topic))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	broker := getEnv("KAFKA_ADDR", "kafka:9092")
	inputTopic := getEnv("KAFKA_TOPIC", "dotnet-forwarder-orders")
	outputTopic := getEnv("KAFKA_OUTPUT_TOPIC", "go-forwarder-orders")

	if err := waitForTopic(ctx, logger, broker, inputTopic); err != nil {
		return err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{broker},
		Topic:       inputTopic,
		GroupID:     "go-forwarder",
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(broker),
		Topic:                  outputTopic,
		RequiredAcks:           kafka.RequireAll,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	defer writer.Close()

	logger.Info("go-forwarder started", slog.String("input_topic", inputTopic), slog.String("output_topic", outputTopic))
	for {
		msg, err := reader.ReadMessage(ctx)
		if errors.Is(err, context.Canceled) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read Kafka message: %w", err)
		}
		logger.InfoContext(ctx, "forward_order", slog.String("key", string(msg.Key)))
		time.Sleep(processingDelay())
		if err := forward(ctx, writer, msg); err != nil {
			return fmt.Errorf("write Kafka message: %w", err)
		}
	}
}

func forward(ctx context.Context, writer *kafka.Writer, msg kafka.Message) error {
	for {
		err := writer.WriteMessages(ctx, kafka.Message{Key: msg.Key, Value: msg.Value, Headers: msg.Headers})
		if !errors.Is(err, kafka.UnknownTopicOrPartition) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
