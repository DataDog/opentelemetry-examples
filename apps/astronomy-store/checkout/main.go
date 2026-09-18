package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultPort     = "8080"
	shutdownTimeout = 10 * time.Second
)

var logger *slog.Logger

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	kafkaTopic := getEnv("KAFKA_TOPIC", "orders")
	kafkaProducer := newKafkaProducer(getEnv("KAFKA_ADDR", "kafka:9092"), kafkaTopic)
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			logger.Error("Kafka producer shutdown failed", slog.Any("error", err))
		}
	}()

	cs := &checkoutServer{
		kafkaProducer: kafkaProducer,
		kafkaTopic:    kafkaTopic,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /checkout/place-order", cs.placeOrder)
	mux.HandleFunc("/checkout/health", cs.health)

	addr := ":" + getEnv("PORT", defaultPort)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		logger.InfoContext(ctx, "shutting down checkout server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown failed", slog.Any("error", err))
		}
	}()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	logger.Info("checkout server listening", slog.String("address", addr))
	if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
