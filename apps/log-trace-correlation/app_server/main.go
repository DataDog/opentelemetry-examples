package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func initLogger() {
	// Use JSON formatter for structured logging.
	// Datadog log pipeline automatically parses JSON logs and extracts
	// dd.trace_id and dd.span_id fields for log-trace correlation.
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func initTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// Create resource with service identity attributes.
	// The semconv package provides standardized attribute keys per OTel spec.
	res, err := resource.New(ctx,
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("log-correlation-go-server"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP gRPC exporter.
	// The endpoint is configured via OTEL_EXPORTER_OTLP_ENDPOINT env var,
	// which points to the Datadog Agent's OTLP receiver.
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create tracer provider with batch span processor for efficient export.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider and propagator.
	// W3C TraceContext + Baggage propagators ensure distributed trace context
	// is propagated across service boundaries via HTTP headers.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

func main() {
	initLogger()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	tp, err := initTracerProvider(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize tracer provider: %v", err)
	}
	defer func() {
		// Allow up to 5 seconds for pending spans to flush on shutdown.
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			log.Errorf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Handler that demonstrates log-trace correlation by extracting the
	// trace context from the incoming request and injecting it into logs.
	injectHandler := func(w http.ResponseWriter, req *http.Request) {
		span := trace.SpanFromContext(req.Context())

		// Log with dd.trace_id and dd.span_id for Datadog log-trace correlation.
		// These fields allow the Datadog backend to link this log entry to the
		// corresponding distributed trace.
		log.WithFields(traceFields(span)).Info("Work is being done in the handler")

		io.WriteString(w, "Log has been injected with trace_id and span_id!\n")
	}

	// Wrap handler with OTel HTTP instrumentation to automatically create
	// server spans and propagate trace context from incoming requests.
	instrumentedHandler := otelhttp.NewHandler(http.HandlerFunc(injectHandler), "Inject")

	mux := http.NewServeMux()
	mux.Handle("/inject", instrumentedHandler)

	// Add a health check endpoint for docker-compose health checks.
	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "ok\n")
	})

	server := &http.Server{
		Addr:              ":3000",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start server in a goroutine so we can handle graceful shutdown.
	go func() {
		log.Info("Server listening on :3000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Block until shutdown signal is received.
	<-ctx.Done()
	log.Info("Shutting down server gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Server shutdown error: %v", err)
	}
}

// traceFields extracts trace context from a span and returns logrus fields
// formatted for Datadog log-trace correlation.
//
// Datadog requires the following JSON log fields for correlation:
//   - "dd.trace_id": the lower 64 bits of the OTel 128-bit trace ID, as a decimal string
//   - "dd.span_id":  the 64-bit span ID, as a decimal string
//
// The Datadog Agent's OTLP ingest pipeline converts 128-bit OTel trace IDs
// to 64-bit by taking the lower 64 bits. Log correlation must use the same
// 64-bit decimal representation so that the Datadog backend can match logs
// to their corresponding traces.
//
// Reference: https://docs.datadoghq.com/tracing/other_telemetry/connect_logs_and_traces/opentelemetry/
func traceFields(span trace.Span) log.Fields {
	sc := span.SpanContext()
	return log.Fields{
		"dd.trace_id": convertTraceID(sc.TraceID().String()),
		"dd.span_id":  convertSpanID(sc.SpanID().String()),
	}
}

// convertTraceID converts a 128-bit hex trace ID to the lower 64-bit decimal
// string that Datadog expects. OTel uses 128-bit trace IDs (32 hex chars),
// but the Datadog backend indexes traces by the lower 64 bits (16 hex chars)
// represented as a decimal number.
func convertTraceID(hexID string) string {
	if len(hexID) < 16 {
		return ""
	}
	// Take the lower 64 bits (last 16 hex characters).
	lower64 := hexID[len(hexID)-16:]
	intValue, err := strconv.ParseUint(lower64, 16, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(intValue, 10)
}

// convertSpanID converts a 64-bit hex span ID to a decimal string
// that Datadog expects for log-trace correlation.
func convertSpanID(hexID string) string {
	if len(hexID) == 0 {
		return ""
	}
	intValue, err := strconv.ParseUint(hexID, 16, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(intValue, 10)
}
