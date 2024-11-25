package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-dd/gameoflife"
	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-dd/logging"
	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var (
	grpcPort = flag.Int("grpcPort", 8081, "Port to be used by the gRPC server")
	httpPort = flag.Int("httpPort", 8082, "Port to be used by the http server")
	logger   *zap.Logger
)

type server struct {
	gameoflifepb.UnimplementedGameOfLifeServer
}

func (s *server) RunGame(ctx context.Context, gameConfiguration *gameoflifepb.GameRequest) (*gameoflifepb.GameResponse, error) {
	logger.Info("Received game configuration", zap.Any("gameConfiguration", gameConfiguration))

	result, err := gameoflife.Run(ctx, gameConfiguration, logger)
	if err != nil {
		logger.Error("Calling gameoflife.Run", zap.Error(err))
		return result, err
	}

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

	tracer.Start(tracer.WithRuntimeMetrics())
	defer tracer.Stop()

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

	s := grpc.NewServer()
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
