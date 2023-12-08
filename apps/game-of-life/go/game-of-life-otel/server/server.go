package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-otel/gameoflife"
	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-otel/logging"
	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	grpcPort = flag.Int("grpcPort", 8081, "Port to be used by the gRPC server")
	httpPort = flag.Int("httpPort", 8082, "Port to be used by the http server")
	logger   *zap.Logger
	tracer   trace.Tracer
)

func InitTracerProvider(ctx context.Context) *sdktrace.TracerProvider {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		logger.Fatal("Constructing new exporter", zap.Error(err))
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}

func InitMeter(ctx context.Context) *metric.MeterProvider {
	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		logger.Fatal("new otlp metric grpc exporter failed: %v", zap.Error(err))
	}
	provider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(exporter)))
	global.SetMeterProvider(provider)
	return provider
}

type server struct {
	gameoflifepb.UnimplementedGameOfLifeServer
}

func (s *server) RunGame(ctx context.Context, gameConfiguration *gameoflifepb.GameRequest) (*gameoflifepb.GameResponse, error) {
	ctx, span := tracer.Start(ctx, "RunGame")
	defer span.End()
	span.SetAttributes(
		attribute.String("rungame_server.request.board", gameConfiguration.Board),
		attribute.Int("rungame_server.request.num_gens", int(gameConfiguration.NumGens)),
	)
	logger = logger.With(
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("span_id", span.SpanContext().SpanID().String()),
	)

	logger.Info("Received game configuration", zap.Any("gameConfiguration", gameConfiguration))

	result, err := gameoflife.Run(ctx, gameConfiguration, logger)
	if err != nil {
		span.RecordError(err)
		logger.Error("Calling gameoflife.Run", zap.Error(err))
		return result, err
	}
	span.SetAttributes(
		attribute.String("rungame_server.response.board", result.Board),
		attribute.String("rungame_server.response.code", result.Code.String()),
	)

	return result, err
}

func main() {
	flag.Parse()
	var err error
	logger, err = logging.NewZapLogger()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	logger = logger.With(zap.String("service", "game-of-life-server"))

	logger.Info("Arguments", zap.Int("grpcPort", *grpcPort), zap.Int("httpPort", *httpPort))

	ctx := context.Background()
	tp := InitTracerProvider(ctx)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}()
	tracer = tp.Tracer("game-of-life-server")

	provider := InitMeter(ctx)
	defer func() {
		ctxTimeout, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		// pushes any last exports to the receiver
		if err = provider.Shutdown(ctxTimeout); err != nil {
			logger.Error("Error shutting down meter provider", zap.Error(err))
		}
	}()
	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		logger.Fatal("failed to start runtime metrics", zap.Error(err))
	}

	// Start HTTP server
	mux := SetupHandlers()

	go func() {
		logger.Info("Starting http server", zap.Int("httpPort", *httpPort))
		if err = http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), mux); err != nil {
			logger.Fatal("HTTP server ended with error", zap.Error(err))
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	logger.Info("Listening on port", zap.Int("grpcPort", *grpcPort))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)
	gameoflifepb.RegisterGameOfLifeServer(s, &server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		logger.Fatal("failed to serve", zap.Error(err))
	}
}

func SetupHandlers() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/readiness", ReadinessHandler)
	mux.HandleFunc("/liveness", LivenessHandler)

	return mux
}

func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	resp := struct {
		Readiness bool `json:"readiness"`
	}{
		Readiness: true,
	}

	json.NewEncoder(w).Encode(resp)
}

func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	resp := struct {
		Liveness bool `json:"liveness"`
	}{
		Liveness: true,
	}

	json.NewEncoder(w).Encode(resp)
}
