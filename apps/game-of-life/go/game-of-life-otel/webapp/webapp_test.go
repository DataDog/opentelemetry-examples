package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-otel/client"
	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func gameRequestToJSONAPI(board string, numGens int32) string {
	return fmt.Sprintf(
		`{"board":"%s", "num_gens":%d}`,
		board,
		numGens,
	)
}

func setupWebapp(t *testing.T) (*tracetest.InMemoryExporter, *client.MockClient, *observer.ObservedLogs) {
	var err error
	core, logs := observer.New(zap.InfoLevel)
	logger = zap.New(core)
	logger = logger.With(zap.String("service", "game-of-life-webapp"))

	exporter := tracetest.NewInMemoryExporter()
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)

	cfg := client.NewClientConfig()
	ctrl := gomock.NewController(t)

	grpcClient := client.NewMockClient(ctrl)
	gomock.InOrder()

	gameOfLifeClient = client.GameOfLifeClientForConnection(cfg, nil, grpcClient, "webapp_test")

	return exporter, grpcClient, logs
}

func sendRequest(payload string, exporter *tracetest.InMemoryExporter) (*httptest.ResponseRecorder, tracetest.SpanStubs) {
	req := httptest.NewRequest(http.MethodPost, "/rungame", strings.NewReader(payload))
	wr := httptest.NewRecorder()
	handler := SetupHandlers()
	handler.ServeHTTP(wr, req)

	return wr, exporter.GetSpans()
}

func checkLogFields(t *testing.T, logs *observer.ObservedLogs, span tracetest.SpanStub) {
	for _, v := range logs.All() {
		numFields := 0
		for _, c := range v.Context {
			switch c.Key {
			case "service":
				assert.Equal(t, "game-of-life-webapp", c.String)
				numFields++
			case "trace_id":
				assert.Equal(t, span.SpanContext.TraceID().String(), c.String)
				numFields++
			case "span_id":
				assert.Equal(t, span.SpanContext.SpanID().String(), c.String)
				numFields++
			}
		}
		assert.Equal(t, 3, numFields)
	}
}

func TestRunGameTrace(t *testing.T) {
	exporter, grpcClient, logs := setupWebapp(t)

	gameRequest := gameoflifepb.GameRequest{
		Board:   "[[1,1],[1,0]]",
		NumGens: 1,
	}

	grpcClient.EXPECT().RunGame(gomock.Any(), gomock.Any(), gomock.Any()).Return(&gameoflifepb.GameResponse{
		Code:  gameoflifepb.ResponseCode_OK,
		Board: "[[1,1],[1,1]]",
	}, nil)

	_, spans := sendRequest(gameRequestToJSONAPI(gameRequest.Board, gameRequest.NumGens), exporter)
	span := spans[0]

	numAttributes := 0
	for _, v := range span.Attributes {
		switch v.Key {
		case "rungame_handler.request.board":
			assert.Equal(t, gameRequest.Board, v.Value.AsString())
			numAttributes++
		case "rungame_handler.request.num_gens":
			assert.Equal(t, int64(gameRequest.NumGens), v.Value.AsInt64())
			numAttributes++
		case "rungame_handler.response.ascii_board":
			assert.Equal(t, "[1 1] \n [1 1] \n ", v.Value.AsString())
			numAttributes++
		case "rungame_client.request.board":
			assert.Equal(t, gameRequest.Board, v.Value.AsString())
			numAttributes++
		case "rungame_client.request.num_gens":
			assert.Equal(t, int64(gameRequest.NumGens), v.Value.AsInt64())
			numAttributes++
		}
	}
	assert.Equal(t, "RunGameHandler", span.Name)
	assert.Len(t, spans, 1)
	assert.Equal(t, 5, numAttributes)
	assert.Len(t, span.Events, 0)

	checkLogFields(t, logs, span)
}

func TestRunGameDecodeErrorTrace(t *testing.T) {
	exporter, _, logs := setupWebapp(t)

	wr, spans := sendRequest("not a json", exporter)
	span := spans[0]

	numAttributes := 0
	for _, v := range span.Attributes {
		switch v.Key {
		case "rungame_handler.request.board":
			numAttributes++
		case "rungame_handler.request.num_gens":
			numAttributes++
		case "rungame_handler.response.ascii_board":
			numAttributes++
		case "rungame_client.request.board":
			numAttributes++
		case "rungame_client.request.num_gens":
			numAttributes++
		}
	}
	assert.Equal(t, "RunGameHandler", span.Name)
	assert.Len(t, spans, 1)
	assert.Equal(t, 0, numAttributes)
	assert.Len(t, span.Events, 1)
	assert.Equal(t, "exception", span.Events[0].Name)
	assert.Equal(t, http.StatusBadRequest, wr.Result().StatusCode)

	checkLogFields(t, logs, span)
}

func TestRunGameInternalErrorTrace(t *testing.T) {
	exporter, grpcClient, logs := setupWebapp(t)

	gameRequest := gameoflifepb.GameRequest{
		Board:   "[[1,1],[1,0]]",
		NumGens: 1,
	}

	grpcClient.EXPECT().RunGame(gomock.Any(), gomock.Any(), gomock.Any()).Return(&gameoflifepb.GameResponse{
		Code: gameoflifepb.ResponseCode_BAD_REQUEST,
	}, errors.New("internal server error"))

	wr, spans := sendRequest(gameRequestToJSONAPI(gameRequest.Board, gameRequest.NumGens), exporter)
	span := spans[0]

	numAttributes := 0
	for _, v := range span.Attributes {
		switch v.Key {
		case "rungame_handler.request.board":
			assert.Equal(t, gameRequest.Board, v.Value.AsString())
			numAttributes++
		case "rungame_handler.request.num_gens":
			assert.Equal(t, int64(gameRequest.NumGens), v.Value.AsInt64())
			numAttributes++
		case "rungame_handler.response.ascii_board":
			numAttributes++
		case "rungame_client.request.board":
			assert.Equal(t, gameRequest.Board, v.Value.AsString())
			numAttributes++
		case "rungame_client.request.num_gens":
			assert.Equal(t, int64(gameRequest.NumGens), v.Value.AsInt64())
			numAttributes++
		}
	}
	assert.Equal(t, "RunGameHandler", span.Name)
	assert.Len(t, spans, 1)
	assert.Equal(t, 4, numAttributes)
	assert.Len(t, span.Events, 1)
	assert.Equal(t, "exception", span.Events[0].Name)
	assert.Equal(t, http.StatusInternalServerError, wr.Result().StatusCode)

	checkLogFields(t, logs, span)
}
