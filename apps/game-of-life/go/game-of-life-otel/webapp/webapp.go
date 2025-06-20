package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-otel/client"
	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-otel/logging"
	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

var (
	httpPort         = flag.Int("httpPort", 8080, "Port for webapp frontend")
	host             = flag.String("host", "localhost:8081", "Host address for gRPC server")
	resources        = flag.String("resources", "webapp/resources", "Filepath of webapp resources folder")
	logger           *zap.Logger
	gameOfLifeClient client.Client
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
	otel.SetMeterProvider(provider)
	return provider
}

func main() {
	flag.Parse()
	var err error
	logger, err = logging.NewZapLogger()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	logger = logger.With(zap.String("service", "game-of-life-webapp"))

	logger.Info("Arguments", zap.String("host", *host), zap.String("resources", *resources))

	ctx := context.Background()
	tp := InitTracerProvider(ctx)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}()

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

	// Set up a connection to the gRPC server
	gameOfLifeClient, err = client.NewGameOfLifeClient(
		"webapp",
		client.WithHost(*host),
	)
	if err != nil {
		logger.Fatal("Did not connect", zap.Error(err))
	}

	// Start HTTP server
	mux := SetupHandlers()

	logger.Info("Starting http server", zap.Int("httpPort", *httpPort))
	if err = http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), mux); err != nil {
		logger.Fatal("HTTP server ended with error", zap.Error(err))
	}
}

// run Runs the game of life program with the given game configuration
func run(ctx context.Context, board string, numGens int32) (*gameoflifepb.GameResponse, error) {
	gameConfig := &gameoflifepb.GameRequest{
		Board:   board,
		NumGens: numGens,
	}
	logger.Info("Running game", zap.Any("gameConfig", gameConfig))
	r, err := gameOfLifeClient.RunGame(ctx, gameConfig)
	if err != nil {
		logger.Error("Calling gameOfLifeClient.RunGame",
			zap.Error(err),
		)
	} else {
		logger.Info("Finished running game", zap.Any("resultBoard", r.GetBoard()))
	}
	return r, err
}

// boardToAscii Converts the given board string to a readable ASCII format
func boardToAscii(board string) (string, error) {
	boardList := make([][]int, 1)
	if err := json.Unmarshal([]byte(board), &boardList); err != nil {
		logger.Error("failed to parse", zap.Error(err))
		return "", err
	}

	result := ""
	for _, v := range boardList {
		result += fmt.Sprintf("%v \n ", v)
	}
	return result, nil
}

func writeError(w http.ResponseWriter, encoder *json.Encoder, code int, err error, message string) {
	w.WriteHeader(code)
	resp := struct {
		Error error `json:"error"`
	}{
		Error: err,
	}
	encoder.Encode(resp)
	logger.Error(message, zap.Error(err))
}

func SetupHandlers() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/readiness", ReadinessHandler)
	mux.HandleFunc("/liveness", LivenessHandler)
	mux.Handle("/rungame", otelhttp.NewHandler(http.HandlerFunc(RunGameHandler), "RunGameHandler"))
	mux.Handle("/", http.FileServer(http.Dir(*resources)))

	mux.HandleFunc("/config.js", ConfigHandler)

	return mux
}

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `window.env = {
		DD_APPLICATION_ID: "%s",
		DD_CLIENT_TOKEN: "%s",
		DD_RUM_PROXY_URL: "%s",
	};`, os.Getenv("DD_APPLICATION_ID"), os.Getenv("DD_CLIENT_TOKEN"), os.Getenv("DD_RUM_PROXY_URL"))
}

func RunGameHandler(w http.ResponseWriter, r *http.Request) {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := otel.Tracer("RunGame").Start(ctx, "RunGameHandler")
	defer span.End()

	var body gameoflifepb.GameRequest
	encoder := json.NewEncoder(w)
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		span.RecordError(err)
		writeError(w, encoder, http.StatusBadRequest, err, "Bad request error")
		return
	}

	logger.Info("Received request", zap.Any("body", &body))
	span.SetAttributes(
		attribute.String("rungame_handler.request.board", body.GetBoard()),
		attribute.Int("rungame_handler.request.num_gens", int(body.GetNumGens())),
	)
	result, err := run(ctx, body.GetBoard(), body.GetNumGens())
	if err != nil {
		writeError(w, encoder, http.StatusInternalServerError, err, "Internal server error")
		return
	}

	ascii, asciiErr := boardToAscii(result.GetBoard())
	if asciiErr != nil {
		writeError(w, encoder, http.StatusBadRequest, asciiErr, "Bad request error")
		return
	}
	w.WriteHeader(http.StatusOK)
	resp := struct {
		ResultBoard string `json:"resultBoard"`
	}{
		ResultBoard: ascii,
	}
	logger.Info("Sending result board",
		zap.Int("httpStatus", http.StatusOK),
		zap.Any("resultBoard", resp.ResultBoard),
	)
	span.SetAttributes(
		attribute.String("rungame_handler.response.ascii_board", ascii),
	)
	encoder.Encode(resp)
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
