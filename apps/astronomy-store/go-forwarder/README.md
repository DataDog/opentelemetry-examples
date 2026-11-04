# Go Forwarder

This Go service built on `kafka-go` is one hop in the [Kafka forwarder chain](../README.md#kafka-forwarder-chain). It consumes `KAFKA_TOPIC` (default `dotnet-forwarder-orders`) and forwards each message, unchanged, to `KAFKA_OUTPUT_TOPIC` (default `go-forwarder-orders`). The broker address comes from `KAFKA_ADDR` (default `kafka:9092`).

Build and deploy it with:

```sh
docker build -t astronomy-store/go-forwarder:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's `inject-sdk` annotation. Consume and produce spans, trace-context propagation, and Data Streams Monitoring checkpoints come from Go auto-instrumentation.

Like `checkout`, the image is built with [Datadog Orchestrion](https://github.com/DataDog/orchestrion), and [`Dockerfile.upstream-otel`](Dockerfile.upstream-otel) builds it with `otelc` for the `upstream` flow.

`kafka-go` keeps no in-process context linking a consumed message to the next produced one, so the service copies the consumed message's headers onto the forwarded message. The `dd-trace-go` producer instrumentation reads the Data Streams pathway from those headers, which keeps `go-forwarder` in the same pathway as the upstream services.
