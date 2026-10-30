# Load Generator

This k6 Deployment continuously runs five virtual users. Each k6 run lasts 24
hours, after which the Deployment starts a replacement Pod so load generation
continues indefinitely. Each iteration calls:

- `GET /api/products` on `frontend`;
- `GET /api/ads` on `frontend`; and
- `POST /api/checkout` on `frontend`, which proxies to `POST /checkout/place-order` on `checkout`.

`fraud-detection` consumes the orders that `checkout` publishes to Kafka, so the load generator does not call it
directly.

A second, low-rate scenario, `suspicious_ad_traffic`, sends 6 requests per minute straight to the `ad` service to
showcase [Datadog App & API Protection](https://docs.datadoghq.com/security/application_security/) detections:

- half are harmless requests that match App & API Protection's built-in detection rules: the `sqlmap/1.7` scanner user
  agent and a benign `<script>alert(1)</script>` string in a query parameter. With App & API Protection enabled, they
  are reported as `attack_attempt.attack_tool` and `attack_attempt.xss`;
- the other half use a wrong Basic auth password and get a `401 Unauthorized` (failed logins).

The thresholds only apply to the regular `astronomy_store` scenario, so the rejected requests don't fail the run.
`AD_URL`, `AD_USERNAME` and `AD_PASSWORD` set the `ad` endpoint and the valid credentials.

Deploy it after the application services:

```sh
kubectl apply -f kubernetes.yaml
```

Set `FRONTEND_URL` in the Deployment to target a different frontend endpoint.
