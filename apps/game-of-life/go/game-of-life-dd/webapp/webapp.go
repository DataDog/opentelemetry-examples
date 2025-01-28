package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-dd/client"
	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-dd/logging"
	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"
	"go.uber.org/zap"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var (
	httpPort         = flag.Int("httpPort", 8080, "Port for webapp frontend")
	host             = flag.String("host", "localhost:8081", "Host address for gRPC server")
	resources        = flag.String("resources", "webapp/resources", "Filepath of webapp resources folder")
	logger           *zap.Logger
	gameOfLifeClient client.Client
)

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

	tracer.Start(tracer.WithRuntimeMetrics())
	defer tracer.Stop()

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
	span, _ := tracer.StartSpanFromContext(ctx, "run")
	defer span.Finish()

	gameConfig := &gameoflifepb.GameRequest{
		Board:   board,
		NumGens: numGens,
	}
	logger.Info("Running game", zap.Any("gameConfig", gameConfig))
	ctx = tracer.ContextWithSpan(ctx, span)
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

// CORS middleware
func corsMiddleware(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Update this for specific origins in production
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next(w, r)
	})
}

func SetupHandlers() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/readiness", corsMiddleware(ReadinessHandler))
	mux.HandleFunc("/liveness", corsMiddleware(LivenessHandler))
	mux.HandleFunc("/rungame", corsMiddleware(RunGameHandler))
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
	spanContext, _ := tracer.Extract(tracer.HTTPHeadersCarrier(r.Header))
	span := tracer.StartSpan("RunGameHandler", tracer.ChildOf(spanContext))
	defer span.Finish()

	ctx := tracer.ContextWithSpan(r.Context(), span)
	var body gameoflifepb.GameRequest
	encoder := json.NewEncoder(w)
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		writeError(w, encoder, http.StatusBadRequest, err, "Bad request error")
		return
	}

	logger.Info("Received request", zap.Any("body", &body))
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
