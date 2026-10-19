# Fraud Detection

This demo consumes the orders that `checkout` publishes to Kafka and logs a random integer `fraud_score` from 0 through 99 for each one. It reads the topic in `KAFKA_TOPIC` (default `orders`) from the broker at `KAFKA_ADDR` (default `kafka:9092`).

Build and deploy it with:

```sh
docker build --build-context shared=../shared -t astronomy-store/fraud-detection:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's Python auto-instrumentation annotation. It does not configure an SDK in the application. For each order, the application extracts the trace context from the Kafka message headers and starts a `process_order` consumer span in `checkout`'s trace, with the `astronomystore.fraud_score` attribute. Logs are emitted as JSON through `structlog`'s standard-library logging integration, which is compatible with OpenTelemetry's logging instrumentation.
