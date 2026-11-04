package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
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

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	broker := getEnv("KAFKA_ADDR", "kafka:9092")
	outputTopic := getEnv("KAFKA_OUTPUT_TOPIC", "go-producer-orders")
	interval, err := time.ParseDuration(getEnv("PRODUCE_INTERVAL", "1s"))
	if err != nil {
		return fmt.Errorf("parse PRODUCE_INTERVAL: %w", err)
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(broker),
		Topic:                  outputTopic,
		RequiredAcks:           kafka.RequireAll,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	defer writer.Close()

	logger.Info("go-producer started", slog.String("output_topic", outputTopic), slog.Duration("interval", interval))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		orderID := uuid.NewString()
		payload, err := json.Marshal(map[string]string{"order_id": orderID, "created_at": time.Now().UTC().Format(time.RFC3339Nano)})
		if err != nil {
			return fmt.Errorf("marshal order: %w", err)
		}
		if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte(orderID), Value: payload}); err != nil {
			logger.ErrorContext(ctx, "failed to produce order", slog.Any("error", err))
			continue
		}
		logger.InfoContext(ctx, "produce_order", slog.String("order_id", orderID))
	}
}
