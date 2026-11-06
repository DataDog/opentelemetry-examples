# Troubleshooting

## OpenTelemetry Operator Internal Metrics

The `cluster` collector scrapes the OpenTelemetry Operator manager's Prometheus metrics with the
`prometheus/otel_operator` receiver (`collectors.cluster.config.receivers` in the values file).

The OpenTelemetry Operator manager exposes the standard [controller-runtime][controller-runtime] metrics, described in
the [Kubebuilder metrics reference][kubebuilder-metrics]. The most useful ones are:

- `controller_runtime_reconcile_errors_total` and `controller_runtime_reconcile_time_seconds` for the reconciliation
  of `OpenTelemetryCollector` and `Instrumentation` resources.
- `workqueue_depth`: a sustained rise means the operator can't keep up.
- `controller_runtime_webhook_latency_seconds`, including the pod-mutating webhook that injects instrumentation.

The metrics endpoint listens on port `8443` and requires the scraper's ServiceAccount token over TLS, with a self-signed
certificate (see [Kubebuilder metrics protection][kubebuilder-metrics-auth]). The `prometheus/otel_operator` scrape
config sets `bearer_token_file` to the Collector's ServiceAccount token and `tls_config.insecure_skip_verify: true`.

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[kubebuilder-metrics]: https://book.kubebuilder.io/reference/metrics-reference
[kubebuilder-metrics-auth]: https://book.kubebuilder.io/reference/metrics#by-using-authnauthz-enabled-by-default
