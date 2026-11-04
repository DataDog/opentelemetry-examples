# Node Forwarder

This Node.js service built on `kafkajs` is one hop in the [Kafka forwarder chain](../README.md#kafka-forwarder-chain). It consumes `KAFKA_TOPIC` (default `java-forwarder-orders`) and forwards each message, unchanged, to `KAFKA_OUTPUT_TOPIC` (default `node-forwarder-orders`). The broker address comes from `KAFKA_ADDR` (default `kafka:9092`).

Build and deploy it with:

```sh
docker build --build-context shared=../shared -t astronomy-store/node-forwarder:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's `inject-nodejs` annotation. Consume and produce spans, trace-context propagation, and Data Streams Monitoring checkpoints come from Node.js auto-instrumentation.
