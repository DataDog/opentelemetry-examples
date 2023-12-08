package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/DataDog/opentelemetry-examples/apps/rpc/protos/calendarpb"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	ocshim "go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	defaultPort = "9090"
	serviceName = "OTEL_SERVICE_NAME"
	PORT_STR    = "PORT"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	logger *zap.Logger
	tracer trace.Tracer
)

func initOpenCensus() error {
	view.SetReportingPeriod(1 * time.Second)
	return view.Register(ocgrpc.DefaultServerViews...)
}

func initOpenTelemetry(res *resource.Resource) (*metric.MeterProvider, error) {
	ctx := context.Background()
	otlpexp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	producer := ocshim.NewMetricProducer()
	reader := metric.NewPeriodicReader(otlpexp, metric.WithInterval(time.Second), metric.WithProducer(producer))
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader), metric.WithResource(res))
	otel.SetMeterProvider(meterProvider)
	return meterProvider, nil
}

// InitTracer creates and registers globally a new TracerProvider.
func InitTracerProvider(ctx context.Context, res *resource.Resource) (trace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		logger.Fatal("Constructing new exporter", zap.Error(err))
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return tp, nil
}

// getEnv gets an environment variable or returns a default value if it is not set
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func realMain() error {
	ctx := context.Background()
	// resource.WithContainer() adds container.id which the agent will leverage to fetch container tags via the tagger.
	res, err := resource.New(ctx, resource.WithContainer())
	if err != nil {
		logger.Fatal("can't create resource", zap.Error(err))
	}

	port := fmt.Sprintf(":%s", getEnv(PORT_STR, defaultPort))
	service := getEnv(serviceName, "calendar-otel")
	logger, err = zap.NewDevelopment(zap.Fields(zap.String("service", service)))
	if err != nil {
		return err
	}

	// Traces
	logger.Info("starting tracer", zap.String("service", service))
	tp, err := InitTracerProvider(ctx, res)
	if err != nil {
		return err
	}
	tracer = tp.Tracer(service)

	// Metrics
	err = initOpenCensus()
	if err != nil {
		logger.Fatal("can't init opencensus", zap.Error(err))
	}
	meterProvider, err := initOpenTelemetry(res)
	if err != nil {
		logger.Fatal("can't init opentelemetry", zap.Error(err))
	}
	defer meterProvider.Shutdown(context.Background())

	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}

	logger.Info("Starting on port ", zap.String("port", port))
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
		grpc.StatsHandler(new(ocgrpc.ServerHandler)),
	)
	reflection.Register(grpcServer)
	s := NewServer()
	calendarpb.RegisterCalendarServiceServer(grpcServer, s)
	healthpb.RegisterHealthServer(grpcServer, s)
	if err = grpcServer.Serve(lis); err != nil {
		return err
	}
	return nil
}
