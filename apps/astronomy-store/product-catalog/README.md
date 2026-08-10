# Product Catalog

This .NET 8 API returns the [OpenTelemetry Demo product catalog data](https://github.com/open-telemetry/opentelemetry-demo/blob/main/src/postgresql/init.sql#L80-L91) from `GET /products`. Retrieve an individual product as JSON with `GET /product/{uid}`, for example `GET /product/OLJCESPC7Z`. Unknown UIDs return `404 Not Found`.

Build and deploy it with:

```sh
docker build -t astronomy-store/product-catalog:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's .NET auto-instrumentation annotation. The application does not configure an OpenTelemetry SDK; the Operator instruments its ASP.NET Core HTTP requests.
