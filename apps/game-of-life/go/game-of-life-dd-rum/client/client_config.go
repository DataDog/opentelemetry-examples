package client

import (
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
)

const (
	defaultHost   = "localhost:8081"
	defaultSource = "sourceUnknown"

	defaultGRpcQueryTimeout = time.Millisecond * 200
	defaultGRpcMaxRetries   = 3
)

var (
	defaultGRpcRetryBackoff = grpc_retry.BackoffExponentialWithJitter(time.Millisecond*5, .1)
)

// ClientConfig holds configurations for the gameoflife grpc client
type ClientConfig struct {
	host             string
	source           string
	gRPCQueryTimeout time.Duration
	gRPCMaxRetries   uint
	gRPCBackoff      grpc_retry.BackoffFunc
}

func (cc *ClientConfig) options() []grpc.CallOption {
	return []grpc.CallOption{
		grpc_retry.WithMax(cc.gRPCMaxRetries),
		grpc_retry.WithPerRetryTimeout(cc.gRPCQueryTimeout),
		grpc_retry.WithBackoff(cc.gRPCBackoff),
	}
}

func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		host:             defaultHost,
		source:           defaultSource,
		gRPCQueryTimeout: defaultGRpcQueryTimeout,
		gRPCMaxRetries:   defaultGRpcMaxRetries,
		gRPCBackoff:      defaultGRpcRetryBackoff,
	}
}

// ClientOption is a function that alters the client config.
type ClientOption func(*ClientConfig)

// WithSource modifies the default source
func WithSource(source string) ClientOption {
	return func(cc *ClientConfig) {
		cc.source = source
	}
}

// WithHost specify host to connect to (won't change if host provided is "")
func WithHost(host string) ClientOption {
	return func(cc *ClientConfig) {
		if host != "" {
			cc.host = host
		}
	}
}

// WithQueryTimeout overrides the query timeout in the configuration.
func WithQueryTimeout(t time.Duration) ClientOption {
	return func(client *ClientConfig) { client.gRPCQueryTimeout = t }
}

// WithGRpcMaxRetries sets the number of times a rsp request should retry before moving onto the next batch
func WithGRpcMaxRetries(r uint) ClientOption {
	return func(cc *ClientConfig) {
		cc.gRPCMaxRetries = r
	}
}

// WithBackoff sets the backoff strategy
func WithBackoff(b grpc_retry.BackoffFunc) ClientOption {
	return func(cc *ClientConfig) {
		cc.gRPCBackoff = b
	}
}
