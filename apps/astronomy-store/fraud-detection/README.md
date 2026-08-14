# Fraud Detection

This demo accepts order checks at `POST /fraud-detection/check-order` and returns a random integer `fraud_score` from 0 through 99.

Build and deploy it with:

```sh
docker build -t astronomy-store/fraud-detection:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's Python auto-instrumentation annotation. It does not configure an SDK in the application; the active HTTP server span created by auto-instrumentation is enriched with `astronomystore.fraud_score`. Logs are emitted as JSON through `structlog`'s standard-library logging integration, which is compatible with OpenTelemetry's logging instrumentation.
