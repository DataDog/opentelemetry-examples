# Fraud Detection

This demo consumes orders published by `checkout` to the Kafka topic `orders` (`KAFKA_TOPIC`, default `orders`) on
the broker at `KAFKA_ADDR` (default `kafka:9092`), as consumer group `fraud-detection`. For each order it logs a
random integer `fraud_score` from 0 through 99. It exposes no HTTP endpoint.

Build and deploy it with:

```sh
docker build -t astronomy-store/fraud-detection:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's Python auto-instrumentation annotation. It does not configure an
SDK in the application; the active Kafka consumer span created by auto-instrumentation is enriched with
`astronomystore.fraud_score`. Logs are emitted as JSON through `structlog`'s standard-library logging integration,
which is compatible with OpenTelemetry's logging instrumentation.

The consumer uses the `confluent-kafka` client so that auto-instrumentation traces message consumption and links it
to the trace that `checkout` started when it published the order, propagated via the Kafka message headers.

The consumer uses `auto.offset.reset=earliest`, matching the
[OpenTelemetry Demo's `fraud-detection` service](https://github.com/open-telemetry/opentelemetry-demo/tree/main/src/fraud-detection),
so a freshly started consumer processes orders that were published before it joined instead of silently skipping
them.
