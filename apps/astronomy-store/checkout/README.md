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

The service returns `201 Created` with a JSON body mirroring `PlaceOrderResponse.order` (`order_id`, `shipping_tracking_id`, `shipping_cost`, `shipping_address`). This demo does not run separate cart/shipping/payment services, so checkout charges a flat shipping rate and does not call out to any downstream service. Build and deploy it with:

```sh
docker build -t astronomy-store/checkout:latest .
kubectl apply -f kubernetes.yaml
```

Unlike Java, Go has no bytecode-injected auto-instrumentation, so the service embeds the OpenTelemetry Go SDK directly (traces and OTLP-bridged logs, see `main.go`) and configures it entirely from `OTEL_*` environment variables (exporter endpoint/protocol, resource attributes, service name). The Deployment uses the OpenTelemetry Operator's `instrumentation.opentelemetry.io/inject-sdk` annotation, which injects those environment variables without a language agent. The annotation points at the `Instrumentation` custom resource by `namespace/name` (`opentelemetry-operator-system/opentelemetry-kube-stack`), matching the other astronomy-store services' `inject-java`/`inject-dotnet` annotations — the bare `"true"` value only works when a default `Instrumentation` resource exists in the pod's own namespace, which is not the case here. Application logs are emitted through [`zap`](https://pkg.go.dev/go.uber.org/zap), bridged to the OTel SDK via [`otelzap`](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelzap), so they are exported as OTLP logs.
