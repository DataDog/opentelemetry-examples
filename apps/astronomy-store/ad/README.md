# Ad

This Spring Boot service accepts `GET /ad/get-ads?context_keys=<key>&context_keys=<key>...`. Its request/response shape mirrors the OpenTelemetry Demo [`AdService`](https://github.com/open-telemetry/opentelemetry-demo/blob/main/pb/demo.proto#L256-L262):

```proto
service AdService {
    rpc GetAds(AdRequest) returns (AdResponse) {}
}
message AdRequest {
    repeated string context_keys = 1;
}
message AdResponse {
    repeated Ad ads = 1;
}
message Ad {
    string redirect_url = 1;
    string text = 2;
}
```

`context_keys` is optional. For each key, `AdController#getAds` runs a SQL query against the `astronomy-db` PostgreSQL database, matching the key against the comma-separated `categories` column of the `catalog.products` table:

```sql
SELECT id, name FROM catalog.products WHERE string_to_array(categories, ',') @> ARRAY[?]
```

The seed data in `astronomy-db`'s `init.sql` covers the `telescopes`, `binoculars`, `books`, and `planetariums` categories. Unrecognized keys, missing keys, or a database with no matching products fall back to a default set of ads. The service returns a JSON array of `Ad` objects:

```json
[
  {
    "redirect_url": "/product/telescope-explorer-150",
    "text": "150mm reflector telescopes are 20% off this week."
  }
]
```

Build and deploy it with:

```sh
docker build -t astronomy-store/ad:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's Java auto-instrumentation annotation, so Spring MVC HTTP requests are instrumented automatically. On top of that, the application depends directly on `io.opentelemetry:opentelemetry-api` and uses it in `AdController` to add `app.ads.context_keys` and `app.ads.count` attributes to the current span, and logs an info-level message for every `getAds` invocation.

The service connects to `astronomy-db` (see `../astronomy-db`) via `DB_URL`, `DB_USERNAME`, and `DB_PASSWORD` environment variables, set in `kubernetes.yaml`.
