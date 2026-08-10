# Frontend

This Next.js service proxies `GET /api/products` to `GET /products` on the `product-catalog` service. The target defaults to `http://product-catalog:8080` and can be overridden with `PRODUCT_CATALOG_URL`.

Build and deploy it with:

```sh
docker build -t astronomy-store/frontend:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's Node.js auto-instrumentation annotation. The outgoing `fetch` call can therefore be traced without application-level SDK configuration.
