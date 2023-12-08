package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

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
			log.Error("Error shutting down tracer provider: ", err)
		}
	}()

	injectHandler := func(w http.ResponseWriter, req *http.Request) {
		span := trace.SpanFromContext(req.Context())
		log.WithFields(log.Fields{
			"dd.trace_id": convertHexId(span.SpanContext().TraceID().String()),
			"dd.span_id":  convertHexId(span.SpanContext().SpanID().String()),
		}).Info("Work is being done in the handler")
		io.WriteString(w, "Log has been injected with trace_id and span_id!\n")
	}
	instrumentedHandler := otelhttp.NewHandler(http.HandlerFunc(injectHandler), "Inject")
	http.Handle("/inject", instrumentedHandler)

	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
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
