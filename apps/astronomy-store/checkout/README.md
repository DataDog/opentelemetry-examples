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
and does not call HTTP services downstream. `POST /checkout/place-order` requires HTTP Basic authentication with a single hardcoded user, `frontend.astronomy-store` / `changeme` (see `auth.go`), sent by the `frontend` service; requests without valid credentials get a `401 Unauthorized`. `/checkout/health` stays anonymous for the Kubernetes probes. See [Datadog App & API Protection](#datadog-app--api-protection) below. Build and deploy it with:

```sh
docker build -t astronomy-store/checkout:latest .
kubectl apply -f kubernetes.yaml
```

The image is built with [Datadog Orchestrion](https://github.com/DataDog/orchestrion) v1.13.1, which injects
`dd-trace-go/v2` instrumentation during compilation via `orchestrion go build` in place of `go build`. The service uses
only libraries supported by Orchestrion: standard-library `net/http` for server spans, `log/slog` for log records, and [
`kafka-go`](https://github.com/segmentio/kafka-go) for Kafka producer spans and trace-context propagation. The service
doesn't create SDK providers or instrumentation wrappers itself. It adds domain-specific attributes (`user.id`,
`store.checkout.order.id`...) to the active HTTP span with the OpenTelemetry API (`trace.SpanFromContext(ctx).SetAttributes(...)`).

> [!WARNING]
> In the Orchestrion-built image, these custom attributes never reach the backend
> ([DataDog/orchestrion#936](https://github.com/DataDog/orchestrion/issues/936)). Orchestrion doesn't bridge the
> OpenTelemetry API to `dd-trace-go`: `trace.SpanFromContext` returns a no-op span, because the HTTP server span is a
> `dd-trace-go` span the OpenTelemetry API can't see. And `otel.GetTracerProvider()` isn't backed by `dd-trace-go`, so
> spans started with `otel.Tracer(...).Start(...)` aren't recorded either. This happens with both OTLP and native
> Datadog export, and is unrelated to the App & API Protection issue below. Log records written with `log/slog` are
> still correlated with traces (`dd.trace_id` / `dd.span_id`).

The Deployment's `instrumentation.opentelemetry.io/inject-sdk` annotation supplies the standard `OTEL_*` exporter and
resource environment variables consumed by `dd-trace-go` running in its OpenTelemetry-compatible ("DDOT") mode. The
annotation identifies the `Instrumentation` resource as `opentelemetry-operator-system/opentelemetry-kube-stack`; a bare
`"true"` would require a default `Instrumentation` resource in the `astronomy-store` namespace.

An alternate [`Dockerfile.upstream-otel`](Dockerfile.upstream-otel) builds the same code with the upstream [
`otelc`](https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation) v1.1.0 compiler instead of Datadog
Orchestrion, instrumenting via the OpenTelemetry Go SDK rather than `dd-trace-go/v2`. `./build-images` publishes it
under the same image name with a `-upstream-otel` tag suffix (e.g. `astronomy-store/checkout:latest-upstream-otel`).

## Datadog App & API Protection

`DD_APPSEC_ENABLED=true` in `kubernetes.yaml` enables [Datadog App & API
Protection](https://docs.datadoghq.com/security/application_security/) in the Orchestrion-built image. The upstream
`otelc` image ignores it. App & API Protection's security engine, `libddwaf`, needs glibc, and dd-trace-go finds
`libdl` via `ldconfig` or `ldd`, which distroless images don't ship. So the image is built with `CGO_ENABLED=1` on
`golang:1.26-bookworm` and runs on `gcr.io/distroless/base-debian12`. With `CGO_ENABLED=0`, App & API Protection
silently stays off (`"appsec":false` in the tracer startup log).

### Known issue: security activity is lost over OTLP

App & API Protection runs and detects attacks (`appsec: up and running` in the startup log), but Datadog doesn't show
them as security activity. The `Instrumentation` resource sets `OTEL_TRACES_EXPORTER=otlp`, so `dd-trace-go` exports
traces over OTLP through the OpenTelemetry Collector. On that path:

- The WAF events arrive as a single raw JSON string attribute, `appsec`, instead of the structured `appsec.*`
  attributes. Datadog derives no `@appsec.security_activity`, `@appsec.rule_id`... facets, so nothing shows up in App &
  API Protection.
- Attack traces aren't force-kept. In a test, only 1 of 15 attack requests was indexed, by `diversity_sampling`.

The Java `ad` service, with the same configuration, works end to end. Its DDOT Java SDK sends traces to the Datadog
Agent over the native protocol, so its attacks show up with structured `appsec.*` attributes, kept by App & API
Protection's retention filter.

To reproduce, send attack payloads straight to `checkout` (the `frontend` doesn't forward client headers or the query
string), then search Datadog spans for `service:checkout @appsec.security_activity:*`:

```sh
kubectl -n astronomy-store port-forward svc/checkout 18083:8080 &
curl -A "sqlmap/1.7" "localhost:18083/checkout/health?q=%3Cscript%3Ealert(1)%3C/script%3E"
```
