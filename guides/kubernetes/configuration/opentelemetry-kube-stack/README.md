# kube-stack values.yaml

Reference `values.yaml` for the [opentelemetry-kube-stack][chart] Helm chart, configured to send Kubernetes telemetry to Datadog.

## What this deploys

The `opentelemetry-kube-stack` chart installs the OpenTelemetry Operator and renders one `OpenTelemetryCollector` CR for each entry under `collectors:`. This values file configures two of them:

- **`cluster`** — a single-replica Deployment responsible for cluster-scope telemetry: scraping kube-state-metrics and watching Kubernetes objects.
- **`daemon`** — a DaemonSet running on every node, responsible for node-scope telemetry (host and kubelet metrics) and for terminating the OTLP endpoint that application workloads send traces, logs, and metrics to.

Optionally, the release installs the **host profiler** collector — a DaemonSet running the OpenTelemetry eBPF profiler on every node and exporting continuous profiles to Datadog. See [Host profiler (optional)](#host-profiler-optional).

## Prerequisites

- A Kubernetes secret named `datadog-secret` with keys `api-key` (required) and `dd-site` (optional; defaults to `datadoghq.com`).
- [cert-manager][cm] installed in the cluster, for the operator's admission webhook.
- Linux nodes with kernel >= 5.10, only for the optional host profiler.

## Quickstart

Run the installer from this directory:

```sh
./install
```

The installer prompts for your Datadog API key and site (the site defaults to `datadoghq.com`), Kubernetes platform, deployment environment, and whether to enable the eBPF host profiler. For EKS, GKE, and AKS, it enables the matching resource-detection preset. For other platforms, it prompts for the Kubernetes cluster name.

It then:

- creates the `opentelemetry-operator-system` namespace and the `datadog-secret` secret;
- installs cert-manager when needed;
- installs or upgrades the OpenTelemetry Kube Stack Helm chart;
- when the host profiler is enabled, enables the `host-profiler` collector in the same release (`--set collectors.host-profiler.enabled=true`) and applies the chosen egress NetworkPolicy (see below); and
- installs or upgrades the Datadog Agent (`ddagent-kube-stack` release, see `dd-agent-values.yaml`).

If you choose to save your credentials, the installer writes them to `.env` with permissions restricted to the file owner. Keep this file out of version control.

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
  --version 0.21.0 \
  --namespace opentelemetry-operator-system \
  --values ./values.yaml \
  --values ./deployment/values.yaml
```

Install the Datadog Agent (as the installer also does). Set `datadog.site` to the site you configured for `datadog-secret`. Set `datadog.clusterName` to the same value the installer prompts for on unmanaged Kubernetes; on EKS, GKE, and AKS the installer leaves it empty, letting the Agent detect the cluster name:

```sh
helm repo add datadog https://helm.datadoghq.com
helm upgrade --install ddagent-kube-stack \
  datadog/datadog \
  --version 3.240.0 \
  --namespace opentelemetry-operator-system \
  --set-string datadog.clusterName=my_k8s_cluster \
  --set-string datadog.apiKeyExistingSecret='datadog-secret' \
  --set-string datadog.site=datadoghq.com \
  -f ./dd-agent-values.yaml
```

Optionally, enable the host profiler collector (or answer `y` to the installer's prompt instead):

```sh
helm upgrade --install opentelemetry-kube-stack \
  open-telemetry/opentelemetry-kube-stack \
  --version 0.21.0 \
  --namespace opentelemetry-operator-system \
  --set collectors.host-profiler.enabled=true \
  --values ./values.yaml \
  --values ./deployment/values.yaml
```

On clusters enforcing NetworkPolicy, also apply one of:

```sh
kubectl apply -f ./host-profiler-network-policy.yaml          # any enforcing CNI
kubectl apply -f ./host-profiler-cilium-network-policy.yaml   # Cilium, FQDN-scoped egress
```

## Host profiler (optional)

The host profiler runs the [Datadog host profiler][dd-host-profiler] (Datadog's own distribution of the [OpenTelemetry eBPF profiler][ebpf-profiler], to which it actively contributes) as a collector DaemonSet on every node and exports profiles to Datadog's OTLP intake.

Enable it by answering `y` to the installer's prompt, or install manually following the [manual installation steps][dd-host-profiler-install]. Unlike the other collectors, its pods need more privileges, which is why it is opt-in.

It is a regular `host-profiler` collector of the single `opentelemetry-kube-stack` release (see `values.yaml`). Two chart features make this possible:

- the `profiling` preset (`collectors.host-profiler.presets.profiling`) declares the `profiling` receiver, the `profiles` pipeline, `hostPID`, the `tracefs` volume, and an unprivileged security context with the eBPF capabilities (`BPF`, `PERFMON`, `SYS_PTRACE`, `SYS_RESOURCE`, `DAC_READ_SEARCH`, `SYSLOG`, `CHECKPOINT_RESTORE`, `IPC_LOCK`). The security context below it hardens it further with a Localhost seccomp profile installed on each node by an init container.
- `inheritDefaultCRConfig: false` excludes the shared `config`, `presets`, `scrape_configs_file`, and `targetAllocator` of `defaultCRConfig` from being merged into this collector; structural defaults (resources, RBAC bindings...) still inherit, and a collector-local `env` list replaces the inherited one (which is why the Datadog credentials are redeclared in `values.yaml`). The `ddot-ebpf` image is a dedicated distribution whose binary does not include the shared components (datadog exporter and extension, `transform`, `resource_detection`, `cumulativetodelta`, `otlp` receiver...); inheriting them would crash the collector at startup.

Prerequisites:

- Linux nodes with kernel >= 5.10 (eBPF profiler requirement);
- Kubernetes >= 1.30 for the container-level `appArmorProfile` field. On older clusters, remove the `appArmorProfile` field from BOTH the collector security context and the `seccomp-installer` init container in `values.yaml`, and add these pod annotations through `podAnnotations` instead:

```yaml
podAnnotations:
  container.apparmor.security.beta.kubernetes.io/otc-container: unconfined
  container.apparmor.security.beta.kubernetes.io/seccomp-installer: unconfined
```

### Migrating from the previous two-release setup

Earlier versions of this guide installed the host profiler as a separate `host-profiler` release of the `opentelemetry-collector` chart. If you still have it, remove it BEFORE enabling the integrated collector — otherwise two host profilers run on every node (duplicate eBPF sampling, symbol upload, and profile export):

```sh
helm uninstall host-profiler --namespace opentelemetry-operator-system
```

The integrated `host-profiler` collector of the `opentelemetry-kube-stack` release replaces it entirely; no configuration is lost.

The egress NetworkPolicies (`host-profiler-network-policy.yaml`, `host-profiler-cilium-network-policy.yaml`) are plain Kubernetes manifests applied with `kubectl apply`. They select the collector pods by their `app.kubernetes.io/name: opentelemetry-kube-stack-host-profiler-collector` label, the naming the OpenTelemetry Operator gives to pods of the `opentelemetry-kube-stack-host-profiler` collector.

## Cluster name detection

For EKS, AKS, and GKE, the installer enables the corresponding resource-detection preset in both collectors. The
OpenTelemetry Collector then automatically populates `k8s.cluster.name`.

 For other Kubernetes platforms, the
installer sets `resourceAttributes.k8s.cluster.name` to the supplied cluster name.

See `examples/` for rendered values and manifests for each deployment type. Regenerate them with `make generate-otel-kube-stack-examples`.

The Datadog Agent installed in step 3 (`ddagent-kube-stack`, `datadog/datadog` chart) has its own base values file, `dd-agent-values.yaml`, and its own examples directory, `examples-datadog-agent/`, following the same pattern — `examples-datadog-agent/default/` mimics the `--set-string` overrides the installer applies on top of `dd-agent-values.yaml`. Regenerate its rendered manifests with `make generate-datadog-agent-examples`.

Run `make generate-examples` to regenerate both at once.

## Resource allocation

Both collectors default to `500m` CPU / `1Gi` memory limits and `200m` CPU / `500Mi` memory requests. Scale up for large clusters.

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.21.0` (host-profiler collector requires the `profiling` preset and `inheritDefaultCRConfig`, introduced in `0.21.0`)
- Collector image `otel/opentelemetry-collector-contrib >= 0.154.0` (pinned in values.yaml under `opentelemetry-operator.manager.collectorImage`)

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/

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
[ebpf-profiler]: https://github.com/open-telemetry/opentelemetry-ebpf-profiler
[dd-host-profiler]: https://github.com/DataDog/datadog-agent/tree/main/cmd/host-profiler
[dd-host-profiler-install]: #install-with-values-files

[kubebuilder-metrics]: https://book.kubebuilder.io/reference/metrics-reference
[kube-rbac-proxy]: https://github.com/brancz/kube-rbac-proxy
[kube-rbac-proxy-auth]: https://github.com/brancz/kube-rbac-proxy#authentication--authorization
