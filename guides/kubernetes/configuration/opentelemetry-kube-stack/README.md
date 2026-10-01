# kube-stack values.yaml

Reference `values.yaml` for the [opentelemetry-kube-stack][chart] Helm chart, configured to send Kubernetes telemetry to Datadog. 

## What this deploys

The `opentelemetry-kube-stack` chart installs the OpenTelemetry Operator and renders one `OpenTelemetryCollector` CR for each entry under `collectors:`. This values file configures two of them:

- **`cluster`** — a single-replica Deployment responsible for cluster-scope telemetry: scraping kube-state-metrics and watching Kubernetes objects.
- **`daemon`** — a DaemonSet running on every node, responsible for node-scope telemetry (host and kubelet metrics) and for terminating the OTLP endpoint that application workloads send traces, logs, and metrics to.

With the `ddot` flow, you can also install the **host profiler** collector — a DaemonSet that runs the OpenTelemetry eBPF profiler on every node and exports profiles to Datadog. See [Host profiler (optional)](#host-profiler-optional).

## Prerequisites

- A Kubernetes secret named `datadog-secret` with keys `api-key` (required) and `dd-site` (optional; defaults to `datadoghq.com`).
- [cert-manager][cm] installed in the cluster, for the operator's admission webhook.
- For the optional host profiler only: Linux nodes with kernel >= 5.10 and Kubernetes >= 1.30.
  The host profiler's `securityContext` uses the container-level `appArmorProfile` field,
  introduced in 1.30.

## Quickstart

Configure the installer with environment variables, then run it from this directory:

| Variable | Description | Default |
|---|---|---|
| `DD_API_KEY` | Datadog API key. | None (required) |
| `DD_INSTALL_FLOW` | Installation flow: `upstream` (OTLP/HTTP exporter and upstream OpenTelemetry SDKs) or `ddot` (Datadog exporter, DDOT SDKs, and the Datadog Agent). | None (required) |
| `DD_SITE` | Datadog site, for example `datadoghq.eu`. | `datadoghq.com` |
| `K8S_CLUSTER_TYPE` | Kubernetes platform: `eks` (EKS), `gcp` (GKE), `aks` (AKS), or `other`. | `other` |
| `K8S_CLUSTER_NAME` | Kubernetes cluster name. For `eks`, `gcp`, and `aks`, overrides the auto-detected name. | Auto-detected for `eks`, `gcp`, and `aks`; unset otherwise |
| `DEPLOYMENT_ENVIRONMENT_NAME` | Value of the `deployment.environment.name` resource attribute. | `production` |
| `DD_HOST_PROFILER_ENABLED` | Enable the eBPF host profiler: `true` or `false`. Requires `DD_INSTALL_FLOW=ddot`. See [Host profiler (optional)](#host-profiler-optional). | `false` |
| `DD_HOST_PROFILER_NETWORK_POLICY` | Host profiler egress NetworkPolicy: `none`, `standard`, or `cilium`. | `none` |

```sh
export DD_API_KEY="<your-datadog-api-key>"
export DD_INSTALL_FLOW=upstream
export K8S_CLUSTER_TYPE=eks
./install
```

Alternatively, copy `.env.example` to `.env`, fill it in, and load it before running the installer:

```sh
cp .env.example .env
# Edit .env
source .env
./install
```

For EKS, GKE, and AKS, the installer enables the matching resource-detection preset.

It then:

- creates the `opentelemetry-operator-system` namespace and the `datadog-secret` secret;
- installs cert-manager when needed;
- installs or upgrades the OpenTelemetry Kube Stack Helm chart with `values-otlp-http.yaml` (`upstream`) or `values-ddot.yaml` (`ddot`); and
- for `ddot`, installs or upgrades the Datadog Agent Helm chart with `dd-agent-values.yaml`, and optionally enables the host profiler.

## Install with values files

To perform the same installation without the interactive script, create a Kubernetes secret for the Datadog credentials,
install cert-manager, then apply a platform-specific values overlay.

Set the Datadog credentials:

```sh
export DD_API_KEY="<your-datadog-api-key>"
export DD_SITE="datadoghq.com" # Use your Datadog site when different.
```

Create the namespace and secret consumed by the collectors:

```sh
kubectl create namespace opentelemetry-operator-system \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic datadog-secret \
  --namespace opentelemetry-operator-system \
  --from-literal="api-key=$DD_API_KEY" \
  --from-literal="dd-site=$DD_SITE" \
  --dry-run=client -o yaml | kubectl apply -f -
```

Install cert-manager:

```sh
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true \
  --wait \
  --timeout 5m
```

Create `deployment/values.yaml` by copying the example for the cluster platform, then set its deployment environment.
EKS, GKE, and AKS enable the appropriate resource detector and automatically determine `k8s.cluster.name`:

```sh
mkdir -p deployment

# Choose one:
cp examples/eks-deployment/values.yaml deployment/values.yaml
cp examples/gcp-deployment/values.yaml deployment/values.yaml
cp examples/aks-deployment/values.yaml deployment/values.yaml
```

For other Kubernetes platforms, start with the manual cluster-name example and replace `my_k8s_cluster` and `production`
with the cluster name and deployment environment. `DD_SITE` continues to be sourced from `datadog-secret`.

```sh
mkdir -p deployment
cp examples/manually-set-k8s-cluster-name/values.yaml deployment/values.yaml
```

The selected file is an overlay for this directory's base `values.yaml`; add any deployment-specific configuration to
`./deployment/values.yaml`. Install or upgrade the chart with both files:

```sh
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
helm upgrade --install opentelemetry-kube-stack \
  open-telemetry/opentelemetry-kube-stack \
  --version 0.20.8 \
  --namespace opentelemetry-operator-system \
  --values ./values.yaml \
  --values ./deployment/values.yaml
```

## Host profiler (optional)

The host profiler is available only with the `ddot` flow, which uses the `opentelemetry-kube-stack` chart `0.21.0`. This collector runs the [Datadog host profiler][dd-host-profiler], Datadog's distribution of the [OpenTelemetry eBPF profiler][ebpf-profiler], as a DaemonSet on every node and exports profiles to Datadog's OTLP intake.

The host profiler is opt-in because its pods need more privileges than the other collectors. To enable it, set `DD_HOST_PROFILER_ENABLED=true` before you run the installer. The installer then also applies `host-profiler-rbac-values.yaml` and, on clusters that enforce NetworkPolicy, the policy that `DD_HOST_PROFILER_NETWORK_POLICY` selects: `host-profiler-network-policy.yaml` (`standard`, any enforcing CNI) or `host-profiler-cilium-network-policy.yaml` (`cilium`, FQDN-scoped egress).

## Cluster name detection

For EKS, AKS, and GKE, the installer enables the corresponding resource-detection preset in both collectors. The
OpenTelemetry Collector then automatically populates `k8s.cluster.name`, unless `K8S_CLUSTER_NAME` is set.

 For other Kubernetes platforms, the
installer sets `resourceAttributes.k8s.cluster.name` to `K8S_CLUSTER_NAME` if that variable is set.

See `examples/` for rendered values and manifests for each deployment type.

## Resource allocation

Both collectors default to `500m` CPU / `1Gi` memory limits and `200m` CPU / `500Mi` memory requests. Scale up for large clusters.

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.20.8`
- Collector image `otel/opentelemetry-collector-contrib >= 0.154.0` (pinned in values.yaml under `opentelemetry-operator.manager.collectorImage`)

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/
[dd-host-profiler]: https://github.com/DataDog/datadog-agent/tree/main/cmd/host-profiler
[ebpf-profiler]: https://github.com/open-telemetry/opentelemetry-ebpf-profiler

## Appendix

### OpenTelemetry Operator Internal Metrics

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
  asked to look at that object's current state and drive the cluster's actual
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

