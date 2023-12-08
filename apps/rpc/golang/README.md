# Calendar GRPC server

Generate proto files using script `generate.sh`

`calendar-dd` is instrumented with DD tracer.
`calendar-otel` is instrumented with Otel tracer.

Both the servers have implemented [health grpc](https://github.com/grpc-ecosystem/grpc-health-probe)

## kubernetes

***Build***

calendar-dd

```
docker build --file Dockerfile.calendar.go.dd --tag otel-demo:calendar-go-dd .
```

calendar-otel

```
docker build --file Dockerfile.calendar.go.otel --tag otel-demo:calendar-go-otel .
```

***Deploy***

Install calendar-dd

```
helm install -n otel-ingest calendar-go-dd-ingest ./calendar-dd/k8s --set node_group=ng-1
```


Install calendar-otel

```
helm install -n otel-ingest calendar-go-otel-ingest ./calendar-otel/k8s/ --set node_group=ng-1
```


# Calendar client

When you have the calendar server running locally, you can use calendar-client to send gRPC requests to the server.

```
cd calendar-client 
go run main.go
```

The client is instrumented with [OpenCensus gRPC plug-in](https://pkg.go.dev/go.opencensus.io/plugin/ocgrpc) which automatically collects gRPC metrics. It then uses the [OpenCensus Bridge from OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go/tree/main/bridge/opencensus) and [OTLP exporter](https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp/otlpmetric) to export the metrics in OTLP format. 
