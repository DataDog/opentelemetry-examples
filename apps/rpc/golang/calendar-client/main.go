package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/DataDog/opentelemetry-examples/apps/rpc/protos/calendarpb"

	// NOTE: OpenCensus is deprecated (archived since 2023). The OpenCensus gRPC
	// plugin is used here to demonstrate the OpenCensus-to-OTel bridge for
	// collecting gRPC client metrics (e.g., grpc.io/client/roundtrip_latency).
	// Migration path: Replace ocgrpc.ClientHandler with otelgrpc.NewClientHandler()
	// which provides native OTel metrics in otelgrpc v0.49.0+.
	// See: https://opentelemetry.io/docs/migration/opencensus/
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"

	"go.opentelemetry.io/otel"
	ocshim "go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	defaultServerAddr = "localhost:9090"
	serverAddrStr     = "SERVER_ADDR"
)

// getEnv gets an environment variable or returns a default value if it is not set.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

// initOpenCensus registers default OpenCensus gRPC client metric views.
// These metrics (e.g., grpc.io/client/roundtrip_latency, grpc.io/client/sent_bytes_per_rpc)
// are bridged to OTel and exported to the Datadog Agent's OTLP intake.
//
// Deprecated: OpenCensus is archived. Migrate to otelgrpc.NewClientHandler().
func initOpenCensus() error {
	view.SetReportingPeriod(1 * time.Second)
	return view.Register(ocgrpc.DefaultClientViews...)
}

// initOpenTelemetry sets up the OTel MeterProvider with two readers:
// 1. OTLP gRPC exporter - sends metrics to Datadog Agent (default endpoint: 0.0.0.0:4317)
// 2. Stdout exporter - prints metrics to console for debugging
// Both readers include the OpenCensus bridge producer for bridging ocgrpc metrics.
func initOpenTelemetry() (*metric.MeterProvider, error) {
	ctx := context.Background()
	// Export metrics via OTLP gRPC to the Datadog Agent's OTLP intake.
	// The Datadog Agent maps OTLP metrics to Datadog custom metrics.
	otlpexp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure(), otlpmetricgrpc.WithEndpoint("0.0.0.0:4317"))
	if err != nil {
		return nil, err
	}
	stdexp, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}
	// Bridge OpenCensus metrics (from ocgrpc) into the OTel pipeline.
	producer := ocshim.NewMetricProducer()
	reader1 := metric.NewPeriodicReader(otlpexp, metric.WithInterval(time.Second), metric.WithProducer(producer))
	reader2 := metric.NewPeriodicReader(stdexp, metric.WithInterval(time.Second), metric.WithProducer(producer))
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader1), metric.WithReader(reader2))
	otel.SetMeterProvider(meterProvider)
	return meterProvider, nil
}

func main() {
	err := initOpenCensus()
	if err != nil {
		log.Fatal(err)
	}

	meterProvider, err := initOpenTelemetry()
	if err != nil {
		log.Fatal(err)
	}
	defer meterProvider.Shutdown(context.Background())

	serverAddr := getEnv(serverAddrStr, defaultServerAddr)
	// Use grpc.NewClient instead of the deprecated grpc.Dial.
	// grpc.NewClient creates a ClientConn with lazy connection establishment.
	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(new(ocgrpc.ClientHandler)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := calendarpb.NewCalendarServiceClient(conn)
	resp, err := client.GetRandomDate(context.Background(), &calendarpb.GetDateRequest{})
	if err != nil {
		// Log gRPC status code for better error diagnostics.
		st, ok := status.FromError(err)
		if ok {
			log.Fatalf("gRPC error: code=%s message=%s", st.Code(), st.Message())
		}
		log.Fatal(err)
	}
	fmt.Println("GetRandomDate: ", resp)

	// Allow time for metrics to be flushed before exiting.
	time.Sleep(time.Second * 2)
}
