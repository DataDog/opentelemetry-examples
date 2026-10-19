# Checkout

This Go service accepts a checkout request at `POST /checkout/place-order`, mirroring the [`CheckoutService.PlaceOrder`](https://github.com/open-telemetry/opentelemetry-demo/blob/main/src/checkout/main.go#L306C21-L420) RPC from the OpenTelemetry Demo, whose language (Go) and [`PlaceOrderRequest`/`PlaceOrderResponse`](https://github.com/open-telemetry/opentelemetry-demo/blob/main/pb/demo.proto#L228-L241) shapes this service reuses. Its JSON request body mirrors the OpenTelemetry Demo `PlaceOrderRequest` structure:

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

The service returns `201 Created` with a JSON body mirroring `PlaceOrderResponse.order` (`order_id`, `shipping_tracking_id`, `shipping_cost`, `shipping_address`). It also publishes the order to the `orders` Kafka topic, with the request's trace context in the message headers. This demo does not run separate cart/shipping/payment services, so checkout charges a flat shipping rate and does not call HTTP services downstream. Build and deploy it with:

```sh
docker build -t astronomy-store/checkout:latest .
kubectl apply -f kubernetes.yaml
```

The image is built with [Datadog Orchestrion](https://github.com/DataDog/orchestrion), which injects `dd-trace-go/v2` instrumentation at compile time. Orchestrion instruments the `net/http` server, `log/slog` logs, and the [`kafka-go`](https://github.com/segmentio/kafka-go) producer, including trace-context propagation. The application creates no SDK providers and only adds its domain-specific attributes to the active HTTP span. The Deployment's `instrumentation.opentelemetry.io/inject-sdk` annotation supplies the standard `OTEL_*` exporter and resource environment variables, which `dd-trace-go` reads in its OpenTelemetry-compatible ("DDOT") mode.

[`Dockerfile.upstream-otel`](Dockerfile.upstream-otel) builds the same code with the upstream [`otelc`](https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation) compiler, which instruments it with the OpenTelemetry Go SDK instead. `./build-images` publishes this image with a `-upstream-otel` tag suffix, for example `astronomy-store/checkout:latest-upstream-otel`. `../install-app` deploys the image that matches `DD_INSTALL_FLOW`, the same variable the OpenTelemetry Kube Stack `install` script uses:

- `upstream`: the `-upstream-otel` image.
- `ddot`: the Orchestrion image.
