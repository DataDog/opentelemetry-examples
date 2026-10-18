# Ad

This Spring Boot service accepts `GET /ad/get-ads?context_keys=<key>&context_keys=<key>...&user_id=<id>&user_email=<email>`. Its request/response shape mirrors the OpenTelemetry Demo [`AdService`](https://github.com/open-telemetry/opentelemetry-demo/blob/main/pb/demo.proto#L256-L262):

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
SELECT id, name FROM catalog.products WHERE ? = ANY(string_to_array(categories, ','))
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

The Deployment uses the OpenTelemetry Operator's Java auto-instrumentation annotation, so Spring MVC HTTP requests are instrumented automatically. On top of that, the application depends directly on `io.opentelemetry:opentelemetry-api` and uses it in `AdController` to add `store.ad.context_keys` and `store.ad.count` attributes to the current span, and logs an info-level message for every `getAds` invocation. Logs are written to stdout as one [Logstash-format](https://docs.spring.io/spring-boot/reference/features/logging.html#features.logging.structured) JSON object per line (`logging.structured.format.console=logstash` in `application.properties`), including the `dd.trace_id` and `dd.span_id` that `dd-java-agent` injects into the MDC.

The service connects to `astronomy-db` (see `../astronomy-db`) via `DB_URL`, `DB_USERNAME`, and `DB_PASSWORD` environment variables, set in `kubernetes.yaml`.

## Authentication

`GET /ad/get-ads` requires [HTTP Basic authentication](https://docs.spring.io/spring-security/reference/servlet/authentication/passwords/basic.html) with a single hardcoded user, `frontend.astronomy-store` / `changeme`, declared with `spring.security.user.name` and `spring.security.user.password` in `application.properties`. The `frontend` service sends these credentials. `GET /ad/health` stays anonymous for the Kubernetes probes. See `SecurityConfig`.

```sh
curl -u frontend.astronomy-store:changeme "http://ad:8080/ad/get-ads?context_keys=telescopes"
```

Authentication goes through Spring Security, so [Datadog App & API Protection](https://docs.datadoghq.com/security/application_security/) can automatically track login success and failure events (user activity, account takeover detection). App & API Protection is enabled with `DD_APPSEC_ENABLED=true` in `kubernetes.yaml`. Requests with wrong credentials get a `401 Unauthorized`.

## Sensitive Data Scanner demo

To demo [Datadog Sensitive Data Scanner](https://docs.datadoghq.com/security/sensitive_data_scanner/), the `frontend`
passes the (simulated) signed-in user's ID and email to `GET /ad/get-ads` as the optional `user_id` and `user_email`
query parameters. `AdController#getAds` records them as `user.id` and `user.email` span attributes and logs them. So the
email address shows up in the URL of the `frontend` HTTP client span and of the `ad` server span, in the `ad` span
attributes, and in the `ad` logs. Passing and logging personal data like this is a bad practice, done on purpose here.
The users are fake, with `example.com` addresses.

To see it in Datadog, enable a Sensitive Data Scanner scanning group covering APM spans and logs of the
`astronomy-store` services, with the "Email Address Scanner" library rule.
