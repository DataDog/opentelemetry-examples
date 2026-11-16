# DSM Pipeline

A Java service built on `kafka-clients` for checking Datadog Data Streams Monitoring (DSM) with OpenTelemetry Java
auto-instrumentation. One image runs as three Deployments, picked by `ROLE`:

```
dsm-producer -> dsm-orders -> dsm-forwarder -> dsm-orders-forwarded -> dsm-consumer
```

- `producer` publishes an order to `KAFKA_OUTPUT_TOPIC` every second.
- `forwarder` consumes `KAFKA_TOPIC`, waits a normally distributed 0–1s processing delay (mean 500ms), and produces
  each message unchanged to `KAFKA_OUTPUT_TOPIC`, so the pathway has an internal service latency.
- `consumer` consumes `KAFKA_TOPIC`.

The broker address comes from `KAFKA_ADDR` (default `kafka:9092`).

Build and deploy it with:

```sh
docker build --build-context shared=../shared -t astronomy-store/dsm-pipeline:latest .
kubectl apply -f kubernetes.yaml
```

Each Deployment uses the OpenTelemetry Operator's `inject-java` annotation. Produce and consume spans, trace-context
propagation, and DSM checkpoints come from Java auto-instrumentation.
