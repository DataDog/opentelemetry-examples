package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultPort        = "8080"
	defaultServiceName = "checkout"
	shutdownTimeout    = 10 * time.Second
)

var logger *zap.Logger

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

// getServiceName reads service.name off the resource, which resource.WithFromEnv()
// populates from OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES injected by the
// OpenTelemetry Operator's instrumentation.opentelemetry.io/inject-sdk annotation.
func getServiceName(res *resource.Resource) string {
	for _, attr := range res.Attributes() {
		if attr.Key == semconv.ServiceNameKey {
			return attr.Value.AsString()
		}
	}
	return defaultServiceName
}

// initTracerProvider builds a TracerProvider whose exporter is selected from
// OTEL_EXPORTER_OTLP_* environment variables, which is how the Operator's
// inject-sdk annotation configures a Go service (Go has no bytecode-injected
// auto-instrumentation, so the SDK must already be embedded in the binary).
func initTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't initialize trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

// initLoggerProvider builds a LoggerProvider from OTEL_EXPORTER_OTLP_* environment
// variables so application logs are emitted through OTLP alongside traces.
func initLoggerProvider(ctx context.Context, res *resource.Resource) (*log.LoggerProvider, error) {
	exporter, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't initialize log exporter: %w", err)
	}
	return log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
		log.WithResource(res),
	), nil
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger, _ = zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	// resource.WithContainer() adds container.id so the Datadog Agent can enrich the
	// telemetry with container tags; resource.WithFromEnv() honors OTEL_SERVICE_NAME
	// and OTEL_RESOURCE_ATTRIBUTES injected by the Operator.
	res, err := resource.New(ctx, resource.WithContainer(), resource.WithFromEnv())
	if err != nil {
		return fmt.Errorf("can't create resource: %w", err)
	}
	serviceName := getServiceName(res)

	loggerProvider, err := initLoggerProvider(ctx, res)
	if err != nil {
		return err
	}
	defer func() {
		if err := loggerProvider.Shutdown(ctx); err != nil {
			logger.Error("loggerProvider shutdown failed", zap.Error(err))
		}
	}()
	// Tee the original stdout core with the OTel bridge core so application logs
	// are emitted to stdout (e.g. for `kubectl logs`) in addition to OTLP.
	logger = zap.New(zapcore.NewTee(logger.Core(), otelzap.NewCore(serviceName, otelzap.WithLoggerProvider(loggerProvider))))

	tracerProvider, err := initTracerProvider(ctx, res)
	if err != nil {
		return err
	}
	defer func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			logger.Error("tracerProvider shutdown failed", zap.Error(err))
		}
	}()

	cs := &checkoutServer{}

	mux := http.NewServeMux()
	mux.Handle("/checkout/place-order", otelhttp.NewHandler(http.HandlerFunc(cs.placeOrder), "PlaceOrder"))
	mux.HandleFunc("/checkout/health", cs.health)

	addr := ":" + getEnv("PORT", defaultPort)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		logger.Info("shutting down checkout server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown failed", zap.Error(err))
		}
	}()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	logger.Info("checkout server listening", zap.String("address", addr))
	if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
