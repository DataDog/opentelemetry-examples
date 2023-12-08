package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/DataDog/opentelemetry-examples/apps/rpc/protos/calendarpb"

	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/otel"
	ocshim "go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultServerAddr = "localhost:9090"
	serverAddrStr     = "SERVER_ADDR"
)

// getEnv gets an environment variable or returns a default value if it is not set
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func initOpenCensus() error {
	view.SetReportingPeriod(1 * time.Second)
	return view.Register(ocgrpc.DefaultClientViews...)
}

func initOpenTelemetry() (*metric.MeterProvider, error) {
	ctx := context.Background()
	otlpexp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure(), otlpmetricgrpc.WithEndpoint("0.0.0.0:4317"))
	if err != nil {
		return nil, err
	}
	stdexp, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}
	producer := ocshim.NewMetricProducer()
	reader1 := metric.NewPeriodicReader(otlpexp, metric.WithInterval(time.Second))
	reader1.RegisterProducer(producer)
	reader2 := metric.NewPeriodicReader(stdexp, metric.WithInterval(time.Second))
	reader2.RegisterProducer(producer)
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
	conn, err := grpc.Dial(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(new(ocgrpc.ClientHandler)),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := calendarpb.NewCalendarServiceClient(conn)
	resp, err := client.GetRandomDate(context.Background(), &calendarpb.GetDateRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("GetRandomDate: ", resp)

	time.Sleep(time.Second * 2)
}
