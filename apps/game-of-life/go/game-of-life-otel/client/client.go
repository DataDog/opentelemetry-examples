package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-otel/logging"
	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"

	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var logger *zap.Logger

//go:generate mockgen -source=$GOFILE -package=$GOPACKAGE -destination=client_mockgen.go

type Client interface {
	RunGame(ctx context.Context, in *gameoflifepb.GameRequest, opts ...grpc.CallOption) (*gameoflifepb.GameResponse, error)
	Close() error
}

type gameOfLifeClient struct {
	source     string
	conn       *grpc.ClientConn
	grpcClient gameoflifepb.GameOfLifeClient
	cfg        *ClientConfig
}

func init() {
	var err error
	logger, err = logging.NewZapLogger()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GameOfLifeClientForConnection(cfg *ClientConfig, conn *grpc.ClientConn, grpcClient gameoflifepb.GameOfLifeClient, source string) *gameOfLifeClient {
	return &gameOfLifeClient{
		conn:       conn,
		grpcClient: grpcClient,
		cfg:        cfg,
		source:     source,
	}
}

// NewGameOfLifeClient creates new client for game of life service.
func NewGameOfLifeClient(source string, options ...ClientOption) (Client, error) {
	logger = logger.With(zap.String("service", "game-of-life-webapp"))
	cfg := NewClientConfig()
	for _, opt := range append([]ClientOption{WithSource(source)}, options...) {
		opt(cfg)
	}

	addr := cfg.host
	logger.Info("Connecting to grpc server", zap.String("grpcAddress", addr))
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "dialing when creating gameoflife client")
	}

	grpcClient := gameoflifepb.NewGameOfLifeClient(conn)
	c := GameOfLifeClientForConnection(cfg, conn, grpcClient, source)
	return c, nil
}

func (c *gameOfLifeClient) RunGame(ctx context.Context, gameRequest *gameoflifepb.GameRequest, opts ...grpc.CallOption) (*gameoflifepb.GameResponse, error) {
	ctx, cancel := prepareContext(ctx, c.source, c.cfg.gRPCQueryTimeout)
	span := trace.SpanFromContext(ctx)
	defer func() {
		cancel()
	}()
	runGameLogger := logger.With(
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("span_id", span.SpanContext().SpanID().String()),
	)

	span.SetAttributes(
		attribute.String("rungame_client.request.board", gameRequest.Board),
		attribute.Int("rungame_client.request.num_gens", int(gameRequest.NumGens)),
	)
	gopts := c.cfg.options()
	r, err := c.grpcClient.RunGame(ctx, gameRequest, gopts...)
	if err != nil {
		runGameLogger.Error("Calling grpcClient.RunGame",
			zap.Error(err),
		)
		span.RecordError(err)
		return r, err
	}
	span.SetAttributes(
		attribute.String("rungame_client.response.board", r.Board),
	)
	return r, err
}

func (c *gameOfLifeClient) Close() error {
	return c.conn.Close()
}

// prepareContext adds timeouts and source metadata to the context
func prepareContext(ctx context.Context, source string, timeout time.Duration) (context.Context, context.CancelFunc) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md.Append("source", source)
	return context.WithTimeout(metadata.NewOutgoingContext(ctx, md), timeout)
}
