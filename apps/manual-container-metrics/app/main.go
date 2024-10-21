package main

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	metric2 "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func initLogger() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func initMeter(ctx context.Context, r *resource.Resource) *metric.MeterProvider {
	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		log.Fatal("new otlp metric grpc exporter failed: %v", zap.Error(err))
	}
	provider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(exporter)), metric.WithResource(r))
	return provider
}

func initTracerProvider(ctx context.Context, r *resource.Resource) *sdktrace.TracerProvider {
	// Create exporter.
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to construct new exporter: ", err)
	}

	// Create tracer provider.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(r),
	)

	// Set tracer provider and propagator.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp
}

func main() {
	initLogger()

	ctx := context.Background()
	// Create resource.
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithContainer())
	if err != nil {
		log.Fatalf("failed to create resource: ", err)
	}
	tp := initTracerProvider(ctx, res)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Error("Error shutting down tracer provider: ", err)
		}
	}()
	mp := initMeter(ctx, res)
	defer func() {
		if err := mp.Shutdown(ctx); err != nil {
			log.Error("Error shutting down meter provider: ", err)
		}
	}()

	meter := mp.Meter(os.Getenv("OTEL_SERVICE_NAME"))
	containerCpuUsage, err := meter.Float64UpDownCounter("container.cpu.usage")
	containerCpuLimit, err := meter.Float64UpDownCounter("container.cpu.limit")
	containerCpuUser, err := meter.Float64UpDownCounter("container.cpu.user")
	containerCpuSystem, err := meter.Float64UpDownCounter("container.cpu.system")
	containerMemoryRss, err := meter.Float64UpDownCounter("container.memory.rss")
	containerMemoryUsage, err := meter.Float64UpDownCounter("container.memory.usage")
	containerMemoryLimit, err := meter.Float64UpDownCounter("container.memory.limit")
	containerIoRead, err := meter.Float64UpDownCounter("container.io.read")
	containerIoWrite, err := meter.Float64UpDownCounter("container.io.write")
	containerNetSent, err := meter.Float64UpDownCounter("container.net.sent")
	containerNetRcvd, err := meter.Float64UpDownCounter("container.net.rcvd")

	attr := metric2.WithAttributes(attribute.String("customer.attribute", "value"))
	containerCpuUsage.Add(ctx, 1, attr)
	containerCpuLimit.Add(ctx, 1, attr)
	containerCpuUser.Add(ctx, 1, attr)
	containerCpuSystem.Add(ctx, 1, attr)
	containerMemoryRss.Add(ctx, 1, attr)
	containerMemoryUsage.Add(ctx, 1, attr)
	containerMemoryLimit.Add(ctx, 1, attr)
	containerIoRead.Add(ctx, 1, attr)
	containerIoWrite.Add(ctx, 1, attr)
	containerNetSent.Add(ctx, 1, attr)
	containerNetRcvd.Add(ctx, 1, attr)

	// Start HTTP server
	mux := SetupHandlers()
	err = http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func SetupHandlers() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/readiness", otelhttp.NewHandler(http.HandlerFunc(ReadinessHandler), "ReadinessHandler"))
	mux.Handle("/liveness", otelhttp.NewHandler(http.HandlerFunc(LivenessHandler), "LivenessHandler"))

	return mux
}

func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	log.WithFields(log.Fields{
		"dd.trace_id": span.SpanContext().TraceID().String(),
		"dd.span_id":  span.SpanContext().SpanID().String(),
		"service":     os.Getenv("OTEL_SERVICE_NAME"),
	}).Info("Work is being done in readiness handler")
	io.WriteString(w, "Log has been injected with trace_id and span_id!\n")

	resp := struct {
		Readiness bool `json:"readiness"`
	}{
		Readiness: true,
	}

	json.NewEncoder(w).Encode(resp)
}

func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	log.WithFields(log.Fields{
		"dd.trace_id": span.SpanContext().TraceID().String(),
		"dd.span_id":  span.SpanContext().SpanID().String(),
		"service":     os.Getenv("OTEL_SERVICE_NAME"),
	}).Info("Work is being done in liveness handler")
	io.WriteString(w, "Log has been injected with trace_id and span_id!\n")

	resp := struct {
		Liveness bool `json:"liveness"`
	}{
		Liveness: true,
	}

	json.NewEncoder(w).Encode(resp)
}
