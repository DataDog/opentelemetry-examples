package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/DataDog/opentelemetry-examples/apps/rpc/protos/calendarpb"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	// dd-trace-go provides an OTel-compatible TracerProvider that sends traces
	// directly to the Datadog Agent via the native DD trace intake (port 8126).
	// This avoids the OTLP pathway and supports DD-specific features like
	// runtime metrics, profiling correlation, and Dynamic Instrumentation.
	ddotel "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentelemetry"
)

const (
	defaultPort = "9090"
	serviceName = "DD_SERVICE"
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
	logger        *zap.Logger
	tracer        trace.Tracer
	metricsClient *statsd.Client
)

// InitTracerProvider creates and registers globally a new DD-backed TracerProvider.
// The dd-trace-go TracerProvider implements the OTel trace.TracerProvider interface
// so that OTel instrumentation (e.g., otelgrpc) produces DD-native traces.
// DD_AGENT_HOST and DD_TRACE_AGENT_PORT control where traces are sent.
func InitTracerProvider(_ context.Context) (trace.TracerProvider, error) {
	tp := ddotel.NewTracerProvider()
	otel.SetTracerProvider(tp)
	// Set W3C TraceContext and Baggage propagators for context propagation.
	// Ensure DD_TRACE_PROPAGATION_STYLE_INJECT=tracecontext and
	// DD_TRACE_PROPAGATION_STYLE_EXTRACT=tracecontext are set to use W3C
	// propagation (the default for dd-trace-go v1.64+).
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return tp, nil
}

// newMetricsClient creates a DogStatsD client that sends custom metrics to the
// Datadog Agent via Unix domain socket. This is the recommended transport for
// containerized environments (lower latency, no UDP packet loss).
func newMetricsClient(_ context.Context) (*statsd.Client, error) {
	socket := "unix:///var/run/datadog/dsd.socket"
	logger.Info("sending metrics", zap.String("host", socket))
	return statsd.New(socket)
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
	port := fmt.Sprintf(":%s", getEnv(PORT_STR, defaultPort))
	service := getEnv(serviceName, "calendar-dd")
	var err error
	logger, err = zap.NewDevelopment(zap.Fields(zap.String("service", service)))
	if err != nil {
		return err
	}
	logger.Info("starting tracer", zap.String("service", service))
	tp, err := InitTracerProvider(context.Background())
	if err != nil {
		return err
	}

	tracer = tp.Tracer(service)

	metricsClient, err = newMetricsClient(context.Background())
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}

	logger.Info("Starting on port ", zap.String("port", port))

	// Use otelgrpc.NewServerHandler() (stats handler) for tracing instead of
	// the deprecated UnaryServerInterceptor/StreamServerInterceptor.
	// The stats handler automatically creates spans for all gRPC methods with
	// semantic convention attributes (rpc.system, rpc.service, rpc.method).
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
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
