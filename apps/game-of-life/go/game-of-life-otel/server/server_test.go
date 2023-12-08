package main

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func startGRPCServer() *bufconn.Listener {
	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	gameoflifepb.RegisterGameOfLifeServer(srv, &server{})
	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()
	return listener
}

func getBufDialer(listener *bufconn.Listener) func(string, time.Duration) (net.Conn, error) {
	return func(url string, t time.Duration) (net.Conn, error) {
		return listener.Dial()
	}
}

func setupServer(t *testing.T) (*tracetest.InMemoryExporter, gameoflifepb.GameOfLifeClient, *observer.ObservedLogs) {
	var err error
	core, logs := observer.New(zap.InfoLevel)
	logger = zap.New(core)
	logger = logger.With(zap.String("service", "game-of-life-server"))

	exporter := tracetest.NewInMemoryExporter()
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("server_test")

	listener := startGRPCServer()
	conn, err := grpc.DialContext(context.Background(), "", grpc.WithDialer(getBufDialer(listener)), grpc.WithInsecure())
	assert.NoError(t, err)

	return exporter, gameoflifepb.NewGameOfLifeClient(conn), logs
}

func checkGrpcSpanAttributes(t *testing.T, grpcSpan tracetest.SpanStub, statusCode int) {
	numAttributes := 0
	for _, v := range grpcSpan.Attributes {
		switch v.Key {
		case "rpc.system":
			assert.Equal(t, "grpc", v.Value.AsString())
			numAttributes++
		case "rpc.service":
			assert.Equal(t, "gameoflifepb.GameOfLife", v.Value.AsString())
			numAttributes++
		case "rpc.method":
			assert.Equal(t, "RunGame", v.Value.AsString())
			numAttributes++
		case "rpc.grpc.status_code":
			assert.Equal(t, int64(statusCode), v.Value.AsInt64())
			numAttributes++
		}
	}
	assert.Equal(t, "gameoflifepb.GameOfLife/RunGame", grpcSpan.Name)
	assert.Equal(t, 4, numAttributes)
	assert.Len(t, grpcSpan.Events, 2)
}

func checkLogFields(t *testing.T, logs *observer.ObservedLogs, span tracetest.SpanStub) {
	for _, v := range logs.All() {
		numFields := 0
		for _, c := range v.Context {
			switch c.Key {
			case "service":
				assert.Equal(t, "game-of-life-server", c.String)
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
	gameRequest := gameoflifepb.GameRequest{
		Board:   "[[1,1],[1,1]]",
		NumGens: 1,
	}
	exporter, client, logs := setupServer(t)
	resp, err := client.RunGame(context.Background(), &gameRequest)
	assert.Nil(t, err)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 2)
	runGameSpan := spans[0]
	numAttributes := 0
	for _, v := range runGameSpan.Attributes {
		switch v.Key {
		case "rungame_server.request.board":
			assert.Equal(t, v.Value.AsString(), gameRequest.Board)
			numAttributes++
		case "rungame_server.request.num_gens":
			assert.Equal(t, v.Value.AsInt64(), int64(gameRequest.NumGens))
			numAttributes++
		case "rungame_server.response.board":
			assert.Equal(t, v.Value.AsString(), resp.Board)
			numAttributes++
		case "rungame_server.response.code":
			assert.Equal(t, v.Value.AsString(), resp.Code.String())
			numAttributes++
		}
	}
	assert.Equal(t, "RunGame", runGameSpan.Name)
	assert.Equal(t, 4, numAttributes)
	assert.Len(t, runGameSpan.Events, 0)

	grpcSpan := spans[1]
	checkGrpcSpanAttributes(t, grpcSpan, 0)
	checkLogFields(t, logs, runGameSpan)
}

func TestRunGameErrorTrace(t *testing.T) {
	gameRequest := gameoflifepb.GameRequest{
		Board:   "[[1,1],[1,2]]",
		NumGens: 1,
	}
	exporter, client, logs := setupServer(t)
	_, err := client.RunGame(context.Background(), &gameRequest)
	assert.Error(t, err)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 2)
	runGameSpan := spans[0]
	numAttributes := 0
	for _, v := range runGameSpan.Attributes {
		switch v.Key {
		case "rungame_server.request.board":
			assert.Equal(t, v.Value.AsString(), gameRequest.Board)
			numAttributes++
		case "rungame_server.request.num_gens":
			assert.Equal(t, v.Value.AsInt64(), int64(gameRequest.NumGens))
			numAttributes++
		case "rungame_server.response.board":
			numAttributes++
		case "rungame_server.response.code":
			numAttributes++
		}
	}
	assert.Equal(t, "RunGame", runGameSpan.Name)
	assert.Equal(t, 2, numAttributes)
	assert.Len(t, runGameSpan.Events, 1)
	assert.Equal(t, "exception", runGameSpan.Events[0].Name)

	grpcSpan := spans[1]
	checkGrpcSpanAttributes(t, grpcSpan, 2)
	checkLogFields(t, logs, runGameSpan)
}
