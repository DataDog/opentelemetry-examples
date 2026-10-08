# Troubleshooting

## OpenTelemetry Operator Internal Metrics

The `cluster` collector scrapes the operator manager's own Prometheus metrics via the `prometheus/otel_operator`
receiver (`values.yaml`, `collectors.cluster.config.receivers`).

As of v0.154.0 of the OpenTelemetry Operator, like any [controller-runtime][controller-runtime]-based operator, the
manager exposes the standard controller-runtime metrics registry (documented in the
[Kubebuilder metrics reference][kubebuilder-metrics]) on its `--metrics-bind-address` (secured with
`--metrics-secure`, port `8443` by default in this Helm chart, fronted by [kube-rbac-proxy][kube-rbac-proxy]). These are
generic reconciler metrics, not anything OTel-specific — the same set any Kubebuilder/controller-runtime operator emits:

- `controller_runtime_reconcile_total`, `controller_runtime_reconcile_errors_total`,
  `controller_runtime_reconcile_time_seconds` — reconcile loop counts, errors, and latency, per controller. A
  *reconciliation* is one run of a controller's control loop: whenever a watched resource (e.g. an
  `OpenTelemetryCollector` or `Instrumentation` custom resource) is created, updated, or deleted, the controller is
  asked to look at that object's current state and drive the Kubernetes cluster's actual
  state toward the desired state described in the resource spec — creating/updating the Deployment, ConfigMap,
  webhooks, etc. it owns. Reconciliations are also re-run periodically and after transient errors, so these metrics
  are the best signal for whether the operator is keeping up and succeeding.
- `workqueue_depth`, `workqueue_adds_total`, `workqueue_queue_duration_seconds`,
  `workqueue_work_duration_seconds` — health of each controller's *work queue*. Each controller has a work queue that
  decouples "an object changed" (an event from the informer/watch cache) from "process that object" (a
  reconciliation): watch events enqueue the object's key, and a pool of workers dequeues keys one at a time and runs
  the reconcile function for each. This queue also deduplicates rapid-fire updates to the same object and provides
  retry-with-backoff by re-enqueueing keys whose reconciliation failed. `workqueue_depth` is the number of items
  waiting to be processed (a sustained rise means the operator can't keep up), `workqueue_queue_duration_seconds` is
  how long items wait before a worker picks them up, and `workqueue_work_duration_seconds` is how long the actual
  reconcile takes once dequeued.
- `controller_runtime_active_workers` — number of reconcile workers currently running per controller.
- `controller_runtime_webhook_requests_total`, `controller_runtime_webhook_requests_in_flight`,
  `controller_runtime_webhook_latency_seconds` — count, concurrency, and latency of admission webhook calls, labeled
  by `webhook` path (e.g. the pod-mutating webhook that injects instrumentation, and the validating/mutating webhooks
  for the `OpenTelemetryCollector`/`Instrumentation` CRs). Distinct from the reconcile metrics above: webhooks run
  synchronously inside the Kubernetes API request path (at `Pod`/CR admission time), while reconciliation runs
  asynchronously afterwards.
- Standard Go/process runtime metrics (`go_*`, `process_*`).

kube-rbac-proxy sits in front of the metrics endpoint and, for any HTTP client to be let through, requires an
`Authorization: Bearer <token>` header over TLS — it forwards the token to the Kubernetes API server's
TokenReview/SubjectAccessReview endpoints to authenticate the caller and authorize the request (see
[kube-rbac-proxy's authentication/authorization docs][kube-rbac-proxy-auth]). The `prometheus/otel_operator` scrape
config (`values.yaml`, `collectors.cluster.config.receivers`) satisfies this by setting `scheme: https`,
`bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token` (the Collector pod's own ServiceAccount
token, mounted automatically), and `tls_config.insecure_skip_verify: true` since kube-rbac-proxy's default
self-signed serving certificate isn't in the scraper's trust store.

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[kubebuilder-metrics]: https://book.kubebuilder.io/reference/metrics-reference
[kube-rbac-proxy]: https://github.com/brancz/kube-rbac-proxy
[kube-rbac-proxy-auth]: https://github.com/brancz/kube-rbac-proxy#authentication--authorization
