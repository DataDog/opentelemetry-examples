# Frontend

This Next.js service proxies `GET /api/products` to `GET /products` on the `product-catalog` service. The target defaults to `http://product-catalog:8080` and can be overridden with `PRODUCT_CATALOG_URL`.

It also proxies `GET /api/ads?context_keys=...` to `GET /ad/get-ads?context_keys=...` on the `ad` service, forwarding every `context_keys` value. The target defaults to `http://ad:8080` and can be overridden with `AD_URL`. To demo Datadog Sensitive Data Scanner, it also adds the simulated signed-in user's `user_id` and `user_email` (fake `example.com` users picked at random) as query parameters. Requests to `ad` carry HTTP Basic credentials, `frontend.astronomy-store` / `changeme` by default, overridable with `AD_USERNAME` and `AD_PASSWORD`.

It also proxies `POST /api/checkout` to `POST /checkout/place-order` on the `checkout` service, forwarding the request body as-is, the same way the [OpenTelemetry Demo frontend proxies `POST /api/checkout`](https://github.com/open-telemetry/opentelemetry-demo/blob/main/src/frontend/pages/api/checkout.ts) to its checkout service. The target defaults to `http://checkout:8080` and can be overridden with `CHECKOUT_URL`. Requests to `checkout` carry HTTP Basic credentials, `frontend.astronomy-store` / `changeme` by default, overridable with `CHECKOUT_USERNAME` and `CHECKOUT_PASSWORD`.

Build and deploy it with:

```sh
docker build --build-context shared=../shared -t astronomy-store/frontend:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's Node.js auto-instrumentation annotation. The outgoing `fetch` call can therefore be traced without application-level SDK configuration.
