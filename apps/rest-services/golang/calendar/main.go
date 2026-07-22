package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

const (
	defaultPort        = "9090"
	portStr            = "PORT"
	defaultServiceName = "calendar-rest-go"
)

var logger *zap.Logger

func main() {
	if err := realMain(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// getEnv gets an environment variable or returns a default value if it is not set
func getEnv(key, defaultValue string) string {
	// read from environment
	// if not set, return default value
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}

// getServiceName extracts the service name from the resource.
// The resource is created with WithFromEnv() which honors:
// - OTEL_SERVICE_NAME environment variable
// - service.name in OTEL_RESOURCE_ATTRIBUTES
// Falls back to defaultServiceName if not set.
func getServiceName(res *resource.Resource) string {
	for _, attr := range res.Attributes() {
		if attr.Key == semconv.ServiceNameKey {
			return attr.Value.AsString()
		}
	}
	return defaultServiceName
}

func initTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	// autoexport.NewSpanExporter honors OTEL_EXPORTER_OTLP_PROTOCOL (grpc or http/protobuf)
	// and other OTEL_EXPORTER_* environment variables
	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		logger.Fatal("can't initialize trace exporter", zap.Error(err))
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

// deltaTemporalitySelector returns Delta temporality for supported instrument kinds.
// This is the recommended setting for Datadog.
func deltaTemporalitySelector(kind metric.InstrumentKind) metricdata.Temporality {
	switch kind {
	case metric.InstrumentKindCounter,
		metric.InstrumentKindHistogram,
		metric.InstrumentKindObservableGauge,
		metric.InstrumentKindObservableCounter:
		return metricdata.DeltaTemporality
	case metric.InstrumentKindUpDownCounter,
		metric.InstrumentKindObservableUpDownCounter:
		return metricdata.CumulativeTemporality
	}
	panic("unknown instrument kind")
}

func initMetricProvider(ctx context.Context, res *resource.Resource) (*metric.MeterProvider, error) {
	// Create metric exporter based on OTEL_EXPORTER_OTLP_PROTOCOL
	// Default to http/protobuf as recommended by OTel specification
	protocol := strings.ToLower(getEnv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf"))

	var exporter metric.Exporter
	var err error

	switch protocol {
	case "grpc":
		exporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithTemporalitySelector(deltaTemporalitySelector),
		)
	default: // "http/protobuf" or any other value defaults to HTTP
		exporter, err = otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithTemporalitySelector(deltaTemporalitySelector),
		)
	}
	if err != nil {
		return nil, err
	}

	reader := metric.NewPeriodicReader(exporter)

	stdout, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	// Configure exponential histogram for all histogram instruments
	exponentialHistogramView := metric.NewView(
		metric.Instrument{Kind: metric.InstrumentKindHistogram},
		metric.Stream{
			Aggregation: metric.AggregationBase2ExponentialHistogram{
				MaxSize:  160, // Maximum number of buckets
				MaxScale: 20,  // Maximum scale factor
			},
		},
	)

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithReader(metric.NewPeriodicReader(stdout, metric.WithInterval(5*time.Second))),
		metric.WithResource(res),
		metric.WithView(exponentialHistogramView),
	)
	return meterProvider, nil
}

func initLogProvider(ctx context.Context, res *resource.Resource) (*log.LoggerProvider, error) {
	// autoexport.NewLogExporter honors OTEL_EXPORTER_OTLP_PROTOCOL (grpc or http/protobuf)
	// and other OTEL_EXPORTER_* environment variables
	otlpexp, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, err
	}
	stdout, err := stdoutlog.New()
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(otlpexp)),
		log.WithProcessor(log.NewSimpleProcessor(stdout)),
		log.WithResource(res),
	)
	return loggerProvider, nil
}

func setupHandlers(server *Server) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/calendar", otelhttp.NewHandler(http.HandlerFunc(server.calendarHandler), "CalendarHandler"))

	return mux
}

func realMain() error {
	ctx := context.Background()

	endpoint := fmt.Sprintf(":%s", getEnv(portStr, defaultPort))
	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		return err
	}
	// resource.WithContainer() adds container.id which the agent will leverage to fetch container tags via the tagger.
	// resource.WithFromEnv() honors OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES (including service.name)
	res, err := resource.New(ctx, resource.WithContainer(), resource.WithFromEnv())
	if err != nil {
		logger.Fatal("can't create resource", zap.Error(err))
		return err
	}

	// Get service name from resource (honors OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES)
	serviceName := getServiceName(res)
	logger.Info("Using service name", zap.String("service.name", serviceName))

	loggerProvider, err := initLogProvider(ctx, res)
	if err != nil {
		logger.Fatal("can't init opentelemetry", zap.Error(err))
		return err
	}
	defer func() {
		err := loggerProvider.Shutdown(ctx)
		if err != nil {
			logger.Error("loggerProvider Shutdown failed", zap.Error(err))
		}
	}()
	logger = zap.New(otelzap.NewCore(serviceName, otelzap.WithLoggerProvider(loggerProvider)))

	tp, err := initTracerProvider(ctx, res)
	if err != nil {
		return err
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider: ", zap.Error(err))
		}
	}()

	meterProvider, err := initMetricProvider(ctx, res)
	if err != nil {
		logger.Fatal("can't init opentelemetry", zap.Error(err))
		return err
	}
	defer func() {
		err := meterProvider.Shutdown(ctx)
		if err != nil {
			logger.Error("meterProvider Shutdown failed", zap.Error(err))
		}
	}()
	server, err := NewServer(serviceName, meterProvider, tp)
	if err != nil {
		logger.Fatal("can't create new server", zap.Error(err))
		return err
	}

	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	mux := setupHandlers(server)
	logger.Info("Starting server", zap.String("endpoint", endpoint))
	if err := http.Serve(lis, mux); err != nil {
		logger.Fatal("http server has an error ", zap.Error(err))
		return err
	}

	return nil
}
