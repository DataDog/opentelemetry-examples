package main

import (
	"context"
	"fmt"
	"net"
	"os"

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
	ddotel "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentelemetry"
)

const (
	defaultPort = "9090"
	serviceName = "DD_SERVICE"
	PORT_STR    = "PORT"
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

// InitTracer creates and registers globally a new TracerProvider.
func InitTracerProvider(_ context.Context) (trace.TracerProvider, error) {
	tp := ddotel.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return tp, nil
}

func newMetricsClient(_ context.Context) (*statsd.Client, error) {
	socket := "unix:///var/run/datadog/dsd.socket"
	logger.Info("sending metrics", zap.String("host", socket))
	return statsd.New(socket)
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
	port := fmt.Sprintf(":%s", getEnv(PORT_STR, defaultPort))
	service := getEnv(serviceName, "calendar-otel")
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
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
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
