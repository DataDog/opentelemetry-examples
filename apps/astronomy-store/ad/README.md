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
docker build --build-context shared=../shared -t astronomy-store/ad:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's Java auto-instrumentation annotation, so Spring MVC HTTP requests are instrumented automatically. On top of that, the application depends directly on `io.opentelemetry:opentelemetry-api` and uses it in `AdController` to add `app.ads.context_keys` and `app.ads.count` attributes to the current span, and logs an info-level message for every `getAds` invocation.

The service connects to `astronomy-db` (see `../astronomy-db`) via `DB_URL`, `DB_USERNAME`, and `DB_PASSWORD` environment variables, set in `kubernetes.yaml`.

## Authentication

`GET /ad/get-ads` requires [HTTP Basic authentication](https://docs.spring.io/spring-security/reference/servlet/authentication/passwords/basic.html) with a single hardcoded user, `frontend.astronomy-store` / `changeme`, which the `frontend` service sends. `GET /ad/health` stays anonymous for the Kubernetes probes.

```sh
curl -u frontend.astronomy-store:changeme "http://ad:8080/ad/get-ads?context_keys=telescopes"
```

Requests with wrong credentials get a `401 Unauthorized`.

## Sensitive Data Scanner demo

To demo [Datadog Sensitive Data Scanner](https://docs.datadoghq.com/security/sensitive_data_scanner/), the `frontend`
sends the simulated signed-in user's ID and email to `GET /ad/get-ads` as the optional `user_id` and `user_email` query
parameters. The `ad` service records them as `user.id` and `user.email` span attributes and logs them. The email
address shows up in the URLs of the `frontend` HTTP client span and the `ad` server span, in the `ad` span attributes,
and in the `ad` logs. Sending and logging personal data like this is bad practice, done on purpose here. The users are
fake, with `example.com` addresses.

To see the demo in Datadog, enable a Sensitive Data Scanner scanning group that covers APM spans and logs of the
`astronomy-store` services, with the "Email Address Scanner" library rule.

## Feature flags demo

`GET /ad/get-ads` evaluates the `ad-failure` boolean feature flag, modeled on the OpenTelemetry Demo's `adFailure` flag.
When the flag evaluates to `true`, the request fails with a `503 Service Unavailable`. The user ID is the targeting key,
so a user always gets the same variant.

The service uses the vendor-neutral [OpenFeature](https://openfeature.dev/) API with the
[Datadog Feature Flags](https://docs.datadoghq.com/feature_flags/) OpenFeature provider. The
[OpenFeature OpenTelemetry hook](https://github.com/open-feature/java-sdk-contrib/tree/main/hooks/open-telemetry) also
records each flag evaluation as a `feature_flag` event on the current `ad` span, shown in the Logs section of the span
in Jaeger.

### Datadog OpenFeature provider

- The provider gets the flag configurations through the Datadog Agent's Remote Configuration.
- `dd-openfeature` must have the same version as the `dd-java-agent` injected by the OpenTelemetry Operator, the
  `instrumentation.java.image` of the opentelemetry-kube-stack Helm values (1.67.1 as of 2026-10). Bump both together.

To run the demo:

1. In Datadog, go to **Feature Flags** and create a boolean flag with the key `ad-failure` in the environment of the
   `ad` service, serving `false` by default. Enable the flag in that environment. The service doesn't receive disabled
   flags and returns the default value for them.
2. Roll `true` out to some users, for example with a targeting rule "`user_id` is one of `user-1001`" or a percentage
   rollout, which buckets users by user ID. The `frontend` picks one of 4 demo users, `user-1001` to `user-1004`, per
   request.
3. Watch the error rate of the `ad` service rise, and filter its traces with `@feature_flags.ad-failure:true`. See
   [APM trace enrichment](https://docs.datadoghq.com/feature_flags/guide/apm_trace_enrichment/).
4. Turn the flag off. The errors stop with no redeploy.

The load generator's `http_req_failed` threshold is expected to fail while the flag is on.
