# Checkout

This Go service accepts a checkout request at `POST /checkout/place-order`, mirroring the [
`CheckoutService.PlaceOrder`](https://github.com/open-telemetry/opentelemetry-demo/blob/main/src/checkout/main.go#L306C21-L420)
RPC from the OpenTelemetry Demo, whose language (Go) and [`PlaceOrderRequest`/
`PlaceOrderResponse`](https://github.com/open-telemetry/opentelemetry-demo/blob/main/pb/demo.proto#L228-L241) shapes
this service reuses. Its JSON request body mirrors the OpenTelemetry Demo `PlaceOrderRequest` structure:

```json
{
  "user_id": "user-123",
  "user_currency": "USD",
  "address": {
    "street_address": "123 Astronomy Way",
    "city": "Paris",
    "state": "Ile-de-France",
    "country": "FR",
    "zip_code": "75001"
  },
  "email": "user@example.com",
  "credit_card": {
    "credit_card_number": "4111111111111111",
    "credit_card_cvv": 123,
    "credit_card_expiration_year": 2030,
    "credit_card_expiration_month": 12
  }
}
```

The service returns `201 Created` with a JSON body mirroring `PlaceOrderResponse.order` (`order_id`,
`shipping_tracking_id`, `shipping_cost`, `shipping_address`). It also publishes the order as JSON to Kafka's `orders`
topic, including the request trace context in Kafka headers. The test broker creates the topic on the first
publication. This demo does not run separate cart/shipping/payment services, so checkout charges a flat shipping rate
and does not call HTTP services downstream. Build and deploy it with:

```sh
docker build -t astronomy-store/checkout:latest .
kubectl apply -f kubernetes.yaml
```

The image is built with [Datadog Orchestrion](https://github.com/DataDog/orchestrion) v1.13.1, which injects
`dd-trace-go/v2` instrumentation during compilation via `orchestrion go build` in place of `go build`. The service uses
only libraries supported by Orchestrion: standard-library `net/http` for server spans, `log/slog` for log records, and [
`kafka-go`](https://github.com/segmentio/kafka-go) for Kafka producer spans and trace-context propagation. The service
still adds its domain-specific attributes to the active HTTP span, but does not create SDK providers or instrumentation
wrappers itself.

The Deployment's `instrumentation.opentelemetry.io/inject-sdk` annotation supplies the standard `OTEL_*` exporter and
resource environment variables consumed by `dd-trace-go` running in its OpenTelemetry-compatible ("DDOT") mode. The
annotation identifies the `Instrumentation` resource as `opentelemetry-operator-system/opentelemetry-kube-stack`; a bare
`"true"` would require a default `Instrumentation` resource in the `astronomy-store` namespace.

An alternate [`Dockerfile.upstream-otel`](Dockerfile.upstream-otel) builds the same code with the upstream [
`otelc`](https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation) v1.1.0 compiler instead of Datadog
Orchestrion, instrumenting via the OpenTelemetry Go SDK rather than `dd-trace-go/v2`. `./build-images` publishes it
under the same image name with a `-upstream-otel` tag suffix (e.g. `astronomy-store/checkout:latest-upstream-otel`).
