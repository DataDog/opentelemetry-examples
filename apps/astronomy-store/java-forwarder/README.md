# Java Forwarder

This Java service built on `kafka-clients` is one hop in the [Kafka forwarder chain](../README.md#kafka-forwarder-chain). It consumes `KAFKA_TOPIC` (default `python-forwarder-orders`) waits a normally distributed 0–1s processing delay (mean 500ms), and forwards each message, unchanged, to `KAFKA_OUTPUT_TOPIC` (default `java-forwarder-orders`). The broker address comes from `KAFKA_ADDR` (default `kafka:9092`).

Build and deploy it with:

```sh
docker build --build-context shared=../shared -t astronomy-store/java-forwarder:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's `inject-java` annotation. Consume and produce spans, trace-context propagation, and Data Streams Monitoring checkpoints come from Java auto-instrumentation.
