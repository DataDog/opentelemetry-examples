package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func initLogger() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func initTracerProvider(ctx context.Context) *sdktrace.TracerProvider {
	// Create resource.
	res, err := resource.New(ctx, resource.WithContainer())
	if err != nil {
		log.Fatalf("failed to create resource: ", err)
	}

	// Create exporter.
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to construct new exporter: ", err)
	}

	// Create tracer provider.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set tracer provider and propagator.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp
}

func main() {
	initLogger()

	ctx := context.Background()
	tp := initTracerProvider(ctx)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Errorf("Error shutting down tracer provider: %v", err)
		}
	}()
	tracer = tp.Tracer("client-tracer")

	for {
		makeRequest()
		time.Sleep(30 * time.Second)
	}
}

func makeRequest() {
	ctxReq := context.Background()
	childCtx, span := tracer.Start(ctxReq, "BuildRequest")
	defer span.End()

	req, err := http.NewRequestWithContext(childCtx, http.MethodGet, "http://log-correlation-go-server:3000/inject", nil)
	if err != nil {
		log.Fatalf("Failed to create HTTP request: %v", err)
	}

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to execute HTTP request: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("Failed to read HTTP response body: %v", err)
	}

	log.WithFields(log.Fields{
		"dd.trace_id": convertHexId(span.SpanContext().TraceID().String()),
		"dd.span_id":  convertHexId(span.SpanContext().SpanID().String()),
	}).Info("Got the following response: ", string(body))
}

func convertHexId(id string) string {
	if len(id) < 16 {
		return ""
	}
	if len(id) > 16 {
		id = id[16:]
	}
	intValue, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(intValue, 10)
}
