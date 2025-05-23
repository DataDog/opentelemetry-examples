package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

const (
	defaultPort = "9090"
	PORT_STR    = "PORT"
	name        = "calendar-rest-go"
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

func initTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		logger.Fatal("can't initialize grpc trace exporter", zap.Error(err))
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

func deltaSelector(kind metric.InstrumentKind) metricdata.Temporality {
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

func exponentialHistogramSelector(ik metric.InstrumentKind) metric.Aggregation {
	if ik == metric.InstrumentKindHistogram {
		return metric.AggregationBase2ExponentialHistogram{
			MaxSize:  160,
			MaxScale: 20,
		}
	}
	return metric.DefaultAggregationSelector(ik)
}

func initMetricProvider(res *resource.Resource) (*metric.MeterProvider, error) {
	ctx := context.Background()
	otlpexp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithTemporalitySelector(deltaSelector), // ← new!
		otlpmetricgrpc.WithAggregationSelector(exponentialHistogramSelector))
	if err != nil {
		return nil, err
	}
	reader := metric.NewPeriodicReader(otlpexp, metric.WithInterval(time.Second))

	stdout, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithReader(metric.NewPeriodicReader(stdout, metric.WithInterval(5*time.Second))),
		metric.WithResource(res),
	)
	return meterProvider, nil
}

func initLogProvider(res *resource.Resource) (*log.LoggerProvider, error) {
	ctx := context.Background()
	otlpexp, err := otlploggrpc.New(ctx, otlploggrpc.WithInsecure())
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

	endpoint := fmt.Sprintf(":%s", getEnv(PORT_STR, defaultPort))
	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		return err
	}
	// resource.WithContainer() adds container.id which the agent will leverage to fetch container tags via the tagger.
	res, err := resource.New(ctx, resource.WithContainer(), resource.WithFromEnv())
	if err != nil {
		logger.Fatal("can't create resource", zap.Error(err))
		return err
	}

	loggerProvider, err := initLogProvider(res)
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
	logger = zap.New(otelzap.NewCore(name, otelzap.WithLoggerProvider(loggerProvider)))

	tp, err := initTracerProvider(ctx, res)
	if err != nil {
		return err
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider: ", zap.Error(err))
		}
	}()

	meterProvider, err := initMetricProvider(res)
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
	server, err := NewServer(name, meterProvider, tp)
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
