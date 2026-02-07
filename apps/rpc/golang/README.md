# Calendar gRPC Server

Generate proto files using script `generate.sh`

`calendar-dd` is instrumented with the DD tracer (`dd-trace-go` OTel provider).
`calendar-otel` is instrumented with the OTel SDK tracer (OTLP exporter to DD Agent).

Both servers implement [gRPC health checking](https://github.com/grpc-ecosystem/grpc-health-probe).

## gRPC Instrumentation

Both variants use `otelgrpc.NewServerHandler()` (stats handler) for gRPC tracing,
which is the recommended approach as of otelgrpc v0.49.0+. The deprecated
`UnaryServerInterceptor`/`StreamServerInterceptor` pattern has been replaced.

The stats handler automatically creates spans with semantic convention attributes:
- `rpc.system` = "grpc"
- `rpc.service` = the gRPC service name
- `rpc.method` = the gRPC method name
- `rpc.grpc.status_code` = the gRPC status code

These attributes map to Datadog APM resource names when ingested via OTLP.

## OpenCensus Bridge (Deprecated)

The `calendar-client` and `calendar-otel` variants use the OpenCensus gRPC plugin
(`go.opencensus.io/plugin/ocgrpc`) for collecting gRPC metrics (e.g.,
`grpc.io/client/roundtrip_latency`, `grpc.io/server/server_latency`). These metrics
are bridged to OTel via `go.opentelemetry.io/otel/bridge/opencensus` and exported
through the OTLP pipeline to the Datadog Agent.

**OpenCensus is archived (deprecated since 2023).** The recommended migration path is:

1. Replace `ocgrpc.ServerHandler` / `ocgrpc.ClientHandler` with
   `otelgrpc.NewServerHandler()` / `otelgrpc.NewClientHandler()`, which provide
   built-in metrics collection starting with otelgrpc v0.49.0+.
2. Remove the `go.opencensus.io` and `go.opentelemetry.io/otel/bridge/opencensus`
   dependencies.
3. See: https://opentelemetry.io/docs/migration/opencensus/

## Kubernetes

**Build**

calendar-dd

```
docker build --file Dockerfile.calendar.go.dd --tag otel-demo:calendar-go-dd .
```

calendar-otel

```
docker build --file Dockerfile.calendar.go.otel --tag otel-demo:calendar-go-otel .
```

**Deploy**

Install calendar-dd

```
helm install -n otel-ingest calendar-go-dd-ingest ./calendar-dd/k8s --set node_group=ng-1
```


Install calendar-otel

```
helm install -n otel-ingest calendar-go-otel-ingest ./calendar-otel/k8s/ --set node_group=ng-1
```


# Calendar Client

When you have the calendar server running locally, you can use calendar-client to send gRPC requests to the server.

```
cd calendar-client
go run main.go
```

The client is instrumented with [OpenCensus gRPC plug-in](https://pkg.go.dev/go.opencensus.io/plugin/ocgrpc) which automatically collects gRPC metrics. It then uses the [OpenCensus Bridge from OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go/tree/main/bridge/opencensus) and [OTLP exporter](https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp/otlpmetric) to export the metrics in OTLP format.

## TLS Configuration

Both servers currently use insecure connections for simplicity. For production deployments:

1. Generate TLS certificates (e.g., via cert-manager in Kubernetes).
2. Use `grpc.Creds(credentials.NewTLS(tlsConfig))` server option.
3. Use `grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))` client option.
4. See: https://grpc.io/docs/guides/auth/#with-server-authentication-ssltls
