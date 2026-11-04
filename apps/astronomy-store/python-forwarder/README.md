# Python Forwarder

This Python service built on `confluent-kafka` is one hop in the [Kafka forwarder chain](../README.md#kafka-forwarder-chain). It consumes `KAFKA_TOPIC` (default `go-producer-orders`) and forwards each message, unchanged, to `KAFKA_OUTPUT_TOPIC` (default `python-forwarder-orders`). The broker address comes from `KAFKA_ADDR` (default `kafka:9092`).

Build and deploy it with:

```sh
docker build --build-context shared=../shared -t astronomy-store/python-forwarder:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's `inject-python` annotation. Consume and produce spans, trace-context propagation, and Data Streams Monitoring checkpoints come from Python auto-instrumentation.
