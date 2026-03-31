package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDog/opentelemetry-examples/apps/rpc/protos/calendarpb"

	// NOTE: OpenCensus is deprecated (archived since 2023). The OpenCensus bridge
	// (go.opentelemetry.io/otel/bridge/opencensus) is used here to collect gRPC
	// metrics from the ocgrpc stats handler and export them via OTel.
	// Migration path: Replace ocgrpc.ServerHandler with otelgrpc.NewServerHandler()
	// which provides built-in metrics collection in otelgrpc v0.49.0+.
	// See: https://opentelemetry.io/docs/migration/opencensus/
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

	// gracefulShutdownTimeout is the maximum duration to wait for in-flight
	// RPCs to complete during graceful shutdown.
	gracefulShutdownTimeout = 15 * time.Second
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

// initOpenCensus registers the default OpenCensus gRPC server views for metrics
// collection. These views are bridged to OTel via the OpenCensus bridge.
//
// Deprecated: OpenCensus is archived. Migrate to otelgrpc.NewServerHandler()
// which provides native OTel metrics. The OpenCensus bridge is provided for
// backward compatibility only.
func initOpenCensus() error {
	view.SetReportingPeriod(1 * time.Second)
	return view.Register(ocgrpc.DefaultServerViews...)
}

// initOpenTelemetry sets up the OTel MeterProvider with OTLP gRPC exporter
// and the OpenCensus bridge producer for bridging OpenCensus metrics to OTel.
// Metrics are exported to the Datadog Agent's OTLP intake endpoint.
func initOpenTelemetry(res *resource.Resource) (*metric.MeterProvider, error) {
	ctx := context.Background()
	// Export metrics via OTLP gRPC to the Datadog Agent (default: localhost:4317).
	// The agent converts OTLP metrics to Datadog format.
	otlpexp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	// Bridge OpenCensus metrics (from ocgrpc) into the OTel pipeline.
	producer := ocshim.NewMetricProducer()
	reader := metric.NewPeriodicReader(otlpexp, metric.WithInterval(time.Second), metric.WithProducer(producer))
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader), metric.WithResource(res))
	otel.SetMeterProvider(meterProvider)
	return meterProvider, nil
}

// InitTracerProvider creates and registers globally a new TracerProvider.
// Traces are exported via OTLP gRPC to the Datadog Agent which maps OTel
// spans to Datadog APM traces. gRPC span attributes (rpc.system, rpc.service,
// rpc.method, rpc.grpc.status_code) are mapped to Datadog resource names.
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
	// Set W3C TraceContext and Baggage propagators for distributed tracing.
	// The Datadog Agent supports W3C Trace Context propagation natively.
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return tp, nil
}

// getEnv gets an environment variable or returns a default value if it is not set.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func realMain() error {
	ctx := context.Background()
	// resource.WithContainer() adds container.id which the Datadog Agent
	// leverages to fetch container tags via the tagger.
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

	// Metrics: OpenCensus bridge for gRPC metrics
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

	// Use otelgrpc.NewServerHandler() (stats handler) for tracing instead of
	// the deprecated UnaryServerInterceptor/StreamServerInterceptor.
	// See: https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
	//
	// The ocgrpc.ServerHandler stats handler is used in parallel for OpenCensus
	// gRPC metrics, which are bridged to OTel via the OpenCensus bridge.
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.StatsHandler(new(ocgrpc.ServerHandler)),
	)
	reflection.Register(grpcServer)
	s := NewServer()
	calendarpb.RegisterCalendarServiceServer(grpcServer, s)
	healthpb.RegisterHealthServer(grpcServer, s)

	// Graceful shutdown: wait for in-flight RPCs to complete on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received signal, initiating graceful shutdown", zap.String("signal", sig.String()))
		// GracefulStop stops the gRPC server from accepting new connections and
		// RPCs and blocks until all pending RPCs are finished.
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
			logger.Info("graceful shutdown completed")
		case <-time.After(gracefulShutdownTimeout):
			logger.Warn("graceful shutdown timed out, forcing stop")
			grpcServer.Stop()
		}
	}()

	if err = grpcServer.Serve(lis); err != nil {
		return err
	}
	return nil
}
