package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func initTracer() (*sdktrace.TracerProvider, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("trace-generator"),
			semconv.ServiceVersionKey.String("0.1.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := otel.Tracer("trace-generator")

	// HTTP handler that generates a span on each request
	http.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "handle-generate")
		defer span.End()

		// Simulate work
		delay := time.Duration(rand.Intn(50)+10) * time.Millisecond
		time.Sleep(delay)

		span.SetAttributes(
			attribute.String("request.id", fmt.Sprintf("%d", rand.Int63())),
			attribute.Int("response.delay_ms", int(delay.Milliseconds())),
		)

		// Child span
		_, child := tracer.Start(ctx, "compute")
		time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
		child.End()

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Span generated (delay=%v)\n", delay)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	// Background loop that generates spans continuously
	go func() {
		for {
			_, span := tracer.Start(context.Background(), "background-tick")
			span.SetAttributes(attribute.String("tick.time", time.Now().Format(time.RFC3339)))
			time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)
			span.End()
			time.Sleep(5 * time.Second)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("trace-generator listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
