package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// metricCollectionInterval is the interval at which metrics are collected and exported.
	// 10 seconds aligns with the default Datadog Agent metric collection interval.
	metricCollectionInterval = 10 * time.Second

	// serverPort is the port the HTTP server listens on.
	serverPort = ":3000"

	// shutdownTimeout is the maximum time to wait for graceful shutdown.
	shutdownTimeout = 5 * time.Second
)

func initLogger() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

// initResource creates an OTel resource with service information and environment-provided attributes.
// Resource attributes are used by the Datadog exporter for tagging and container correlation.
func initResource(ctx context.Context) (*resource.Resource, error) {
	serviceName := getEnvOrDefault("OTEL_SERVICE_NAME", "manual-container-metrics-app")
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
}

// initMeterProvider creates and configures the OTel MeterProvider with OTLP gRPC export.
// The periodic reader interval is set to align with Datadog Agent collection intervals.
func initMeterProvider(ctx context.Context, res *resource.Resource) (*metric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}
	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(metricCollectionInterval),
		)),
		metric.WithResource(res),
	)
	return provider, nil
}

// initTracerProvider creates and configures the OTel TracerProvider with OTLP gRPC export.
func initTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// registerContainerMetrics creates and records container metrics that correlate with
// Datadog container monitoring. These metric names map to Datadog's container metric
// namespace so that trace-to-container-metrics correlation works in the Datadog UI.
//
// Datadog metric mapping (OTel -> DD):
//   - container.cpu.usage   (Gauge) -> container.cpu.usage
//   - container.cpu.limit   (Gauge) -> container.cpu.limit
//   - container.cpu.user    (Gauge) -> container.cpu.user
//   - container.cpu.system  (Gauge) -> container.cpu.system
//   - container.memory.rss  (Gauge) -> container.memory.rss
//   - container.memory.usage (Gauge) -> container.memory.usage
//   - container.memory.limit (Gauge) -> container.memory.limit
//   - container.io.read     (Gauge) -> container.io.read
//   - container.io.write    (Gauge) -> container.io.write
//   - container.net.sent    (Gauge) -> container.net.sent
//   - container.net.rcvd    (Gauge) -> container.net.rcvd
//
// Note: These metrics use Float64Gauge because they represent point-in-time measurements
// of container resource usage, not cumulative counters. In the OTel spec, Gauge is the
// correct instrument for non-additive values that represent current state.
// UpDownCounter was previously used but is semantically incorrect for these measurements
// because container resource values are absolute readings, not deltas.
//
// The container.name and container.id attributes are required for Datadog to correlate
// these metrics with the correct container in the trace view.
func registerContainerMetrics(ctx context.Context, meter otelmetric.Meter) error {
	containerName := getEnvOrDefault("OTEL_CONTAINER_NAME", "manual-container-metrics-app")
	containerID := os.Getenv("OTEL_K8S_CONTAINER_ID")
	if containerID == "" {
		log.Warn("OTEL_K8S_CONTAINER_ID is not set; container metrics correlation may not work")
	}

	// Attributes required for Datadog container metrics correlation.
	// container.name and container.id must match the actual container for DD to correlate.
	attrs := otelmetric.WithAttributes(
		attribute.String("container.name", containerName),
		attribute.String("container.id", containerID),
	)

	// CPU metrics -- these are point-in-time gauge values representing current CPU state.
	// DD maps these to container.cpu.* metrics in the container monitoring view.
	type metricDef struct {
		name string
		unit string
		desc string
	}

	gaugeMetrics := []metricDef{
		{"container.cpu.usage", "ns", "Total CPU usage of the container in nanoseconds"},
		{"container.cpu.limit", "{cpus}", "CPU limit assigned to the container"},
		{"container.cpu.user", "ns", "User CPU time consumed by the container in nanoseconds"},
		{"container.cpu.system", "ns", "System CPU time consumed by the container in nanoseconds"},
		{"container.memory.rss", "By", "Resident set size (RSS) memory of the container in bytes"},
		{"container.memory.usage", "By", "Total memory usage of the container in bytes"},
		{"container.memory.limit", "By", "Memory limit of the container in bytes"},
		{"container.io.read", "By", "Bytes read from disk by the container"},
		{"container.io.write", "By", "Bytes written to disk by the container"},
		{"container.net.sent", "By", "Bytes sent over the network by the container"},
		{"container.net.rcvd", "By", "Bytes received over the network by the container"},
	}

	for _, m := range gaugeMetrics {
		gauge, err := meter.Float64Gauge(m.name,
			otelmetric.WithDescription(m.desc),
			otelmetric.WithUnit(m.unit),
		)
		if err != nil {
			return fmt.Errorf("failed to create gauge %s: %w", m.name, err)
		}
		// Record an initial value so the metric is registered with the collector.
		gauge.Record(ctx, 1, attrs)
	}

	return nil
}

func main() {
	initLogger()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create resource with service identity and environment attributes.
	res, err := initResource(ctx)
	if err != nil {
		log.Fatalf("Failed to create OTel resource: %v", err)
	}

	// Initialize tracer provider.
	tp, err := initTracerProvider(ctx, res)
	if err != nil {
		log.Fatalf("Failed to initialize tracer provider: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			log.Errorf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Initialize meter provider.
	mp, err := initMeterProvider(ctx, res)
	if err != nil {
		log.Fatalf("Failed to initialize meter provider: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if err := mp.Shutdown(shutdownCtx); err != nil {
			log.Errorf("Error shutting down meter provider: %v", err)
		}
	}()

	// Create a meter for container metrics.
	serviceName := getEnvOrDefault("OTEL_SERVICE_NAME", "manual-container-metrics-app")
	meter := mp.Meter(serviceName)

	// Register container metrics for Datadog correlation.
	if err := registerContainerMetrics(ctx, meter); err != nil {
		log.Fatalf("Failed to register container metrics: %v", err)
	}
	log.Info("Container metrics registered successfully")

	// Set up HTTP server with OTel-instrumented handlers.
	mux := setupHandlers()
	server := &http.Server{
		Addr:         serverPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so we can handle graceful shutdown.
	errCh := make(chan error, 1)
	go func() {
		log.Infof("Starting server on %s", serverPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for shutdown signal or server error.
	select {
	case err := <-errCh:
		log.Fatalf("Server error: %v", err)
	case <-ctx.Done():
		log.Info("Received shutdown signal, shutting down gracefully...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Errorf("Error shutting down server: %v", err)
		}
	}
}

// setupHandlers registers HTTP handlers with OTel instrumentation.
func setupHandlers() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/health", otelhttp.NewHandler(http.HandlerFunc(HealthHandler), "HealthHandler"))
	mux.Handle("/readiness", otelhttp.NewHandler(http.HandlerFunc(ReadinessHandler), "ReadinessHandler"))
	mux.Handle("/liveness", otelhttp.NewHandler(http.HandlerFunc(LivenessHandler), "LivenessHandler"))

	return mux
}

// HealthHandler provides a basic health check endpoint for monitoring.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// ReadinessHandler indicates whether the application is ready to receive traffic.
func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	log.WithFields(log.Fields{
		"dd.trace_id": span.SpanContext().TraceID().String(),
		"dd.span_id":  span.SpanContext().SpanID().String(),
		"service":     os.Getenv("OTEL_SERVICE_NAME"),
	}).Info("Readiness check")

	w.Header().Set("Content-Type", "application/json")
	resp := struct {
		Readiness bool `json:"readiness"`
	}{
		Readiness: true,
	}
	json.NewEncoder(w).Encode(resp)
}

// LivenessHandler indicates whether the application is running.
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	log.WithFields(log.Fields{
		"dd.trace_id": span.SpanContext().TraceID().String(),
		"dd.span_id":  span.SpanContext().SpanID().String(),
		"service":     os.Getenv("OTEL_SERVICE_NAME"),
	}).Info("Liveness check")

	w.Header().Set("Content-Type", "application/json")
	resp := struct {
		Liveness bool `json:"liveness"`
	}{
		Liveness: true,
	}
	json.NewEncoder(w).Encode(resp)
}

// getEnvOrDefault returns the value of the environment variable named by key,
// or the provided default value if the variable is not set.
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
