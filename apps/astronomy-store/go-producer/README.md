# Go Producer

This Go service built on `kafka-go` is one hop in the [Kafka forwarder chain](../README.md#kafka-forwarder-chain). It publishes a small JSON order to `KAFKA_OUTPUT_TOPIC` (default `go-producer-orders`) every `PRODUCE_INTERVAL` (default `1s`). The broker address comes from `KAFKA_ADDR` (default `kafka:9092`).

Build and deploy it with:

```sh
docker build -t astronomy-store/go-producer:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's `inject-sdk` annotation. Consume and produce spans, trace-context propagation, and Data Streams Monitoring checkpoints come from Go auto-instrumentation.

Like `checkout`, the image is built with [Datadog Orchestrion](https://github.com/DataDog/orchestrion), and [`Dockerfile.upstream-otel`](Dockerfile.upstream-otel) builds it with `otelc` for the `upstream` flow.
