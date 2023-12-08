package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/DataDog/opentelemetry-examples/apps/rpc/protos/calendarpb"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	*health.Server
	calendarpb.UnimplementedCalendarServiceServer
	healthpb.UnimplementedHealthServer
}

func (s *Server) GetRandomDate(
	ctx context.Context,
	request *calendarpb.GetDateRequest,
) (*calendarpb.GetDateResponse, error) {
	return &calendarpb.GetDateResponse{
		Code:         calendarpb.Code_CODE_OK,
		ErrorMessage: "",
		Date:         getDate(ctx),
	}, nil
}

func (s *Server) Check(
	ctx context.Context,
	in *healthpb.HealthCheckRequest,
) (*healthpb.HealthCheckResponse, error) {
	_ = getDate(ctx)
	return s.Server.Check(ctx, in)
}

func (s *Server) Watch(in *healthpb.HealthCheckRequest, stream healthpb.Health_WatchServer) error {
	_ = getDate(context.Background())
	return s.Server.Watch(in, stream)
}

func getDate(ctx context.Context) string {
	_, span := tracer.Start(ctx, "get_date",
		trace.WithAttributes(attribute.String("extra.key", "extra.value")))
	defer span.End()

	dayOffset := rand.Intn(365)
	startDate := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local)
	day := startDate.AddDate(0, 0, dayOffset)

	d := day.Format(time.DateOnly)
	logger.Info("random date", zap.String("date", d))
	return d
}

func NewServer() *Server {
	return &Server{
		Server: health.NewServer(),
	}
}
