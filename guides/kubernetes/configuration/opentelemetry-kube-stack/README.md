# kube-stack values.yaml

Reference `values.yaml` for the [opentelemetry-kube-stack][chart] Helm chart, configured to send Kubernetes telemetry to Datadog.

## What this deploys

The `opentelemetry-kube-stack` chart installs the OpenTelemetry Operator and renders one `OpenTelemetryCollector` CR for each entry under `collectors:`. This values file configures two of them:

- **`cluster`** — a single-replica Deployment responsible for cluster-scope telemetry: scraping kube-state-metrics and watching Kubernetes objects.
- **`daemon`** — a DaemonSet running on every node, responsible for node-scope telemetry (host and kubelet metrics) and for terminating the OTLP endpoint that application workloads send traces, logs, and metrics to.

Optionally, the release installs the **host profiler** collector - a DaemonSet running the OpenTelemetry eBPF profiler on every node and exporting profiles to Datadog. See [Host profiler (optional)](#host-profiler-optional).

The Datadog Agent itself is installed separately, via the **[Datadog Operator][dd-operator]** (`DatadogAgent` custom resource, `datadog-agent.yaml`), running as a DaemonSet in its own **`datadog` namespace** — the namespace name used throughout Datadog's own documentation and examples. It is deliberately scoped down (cluster checks, orchestrator explorer, KSM core, process collection, log collection, and APM auto-instrumentation are all disabled): the OTel collectors above already handle Kubernetes object/metrics monitoring and log collection, and application SDKs are instrumented via the OpenTelemetry Operator instead of Datadog single-step APM instrumentation. The Datadog Cluster Agent is kept, as the node Agent relies on it for cluster-level metadata. Dynamic Instrumentation (Live Debugger) stays enabled, including the system-probe module needed for Go apps (Linux kernel >= 5.17, see [Live Debugger for Go][dd-live-debugger-go]).

### Hybrid OTel + Datadog APM instrumentation

`datadog-agent.yaml` ships with `features.apm.instrumentation.enabled: false`, since this guide instruments applications via the OpenTelemetry Operator instead. It can be flipped to `true` to let the Datadog Agent auto-instrument workloads too (e.g. while migrating a service from Datadog tracers to OTel SDKs) — but a pod must never be instrumented by both at once.

The OpenTelemetry Operator marks a pod for injection with `instrumentation.opentelemetry.io/inject-*` pod **annotations**. Datadog's admission controller cannot select on annotations: `apm.instrumentation.targets[].podSelector` is a plain Kubernetes label selector, and only ever matches pod **labels**. So to exclude an OTel-instrumented pod template from Datadog auto-instrumentation, add this label to it directly, alongside the OTel annotations:

```yaml
metadata:
  labels:
    admission.datadoghq.com/enabled: "false"
```

That label makes Datadog's admission controller skip the pod outright — no `podSelector` needed.

## Prerequisites

- A Kubernetes secret named `datadog-secret`, with keys `api-key` (required) and `dd-site` (optional; defaults to `datadoghq.com`), **duplicated in both the `opentelemetry-operator-system` namespace** (read by the OTel collectors' Datadog exporter, see `values.yaml`) **and the `datadog` namespace** (read by the `DatadogAgent` custom resource). Two copies are needed because a `DatadogAgent` CR can only reference a secret in its own namespace, and the OTel collectors run in a different namespace.
- [cert-manager][cm] installed in the cluster, for the operator's admission webhook.
- Linux nodes with kernel >= 5.10, only for the optional host profiler. Enabling the
  host-profiler collector also requires Kubernetes >= 1.30: its `securityContext`
  uses the container-level `appArmorProfile` field, introduced in 1.30.

## Quickstart

For a step-by-step walkthrough, including prerequisites, the installer's prompts, verification, and troubleshooting, see the [installation guide](INSTALL.md).

Download and run the installer. You don't need to clone the repository: by default, the installer downloads its configuration files (`values.yaml`, `datadog-agent.yaml`...) from GitHub:

```sh
curl -fsSL -o install https://raw.githubusercontent.com/DataDog/opentelemetry-examples/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/install
chmod +x install
./install
```

Options (see `./install --help`):

- `--local`: use the configuration files next to the `install` script instead of downloading them, for example to test local changes from a clone of this repository;
- `--config-url=<url>`: download the configuration files from another GitHub folder, like `https://github.com/DataDog/opentelemetry-examples/tree/<branch>/guides/kubernetes/configuration/opentelemetry-kube-stack`;
- `<overlay-values.yaml>`: a values file merged on top of `values.yaml`, as a local file, a URL, or a path relative to the configuration folder (for example `examples/export-to-datadog-and-jaeger/values.yaml`).

The installer reads your Datadog API key, site, and optional application key from a `.env` file next to it, or prompts for them when there's none (the site defaults to `datadoghq.com` only when `.env` doesn't set `DD_SITE`; the prompt requires it). It then prompts for your Kubernetes platform, deployment environment, and whether to enable the eBPF host profiler. For EKS, GKE, and AKS, it enables the matching resource-detection preset. For other platforms, it prompts for the Kubernetes cluster name, and also on EKS, GKE, and AKS when you provide an application key, as [Fleet Automation](#fleet-automation-optional) needs it.

It then:

- creates the `opentelemetry-operator-system` and `datadog` namespaces;
- creates the `datadog-secret` secret in both namespaces (see [Prerequisites](#prerequisites) for why it's duplicated);
- installs cert-manager, unless it's already installed (detected by its `certificates.cert-manager.io` CRD);
- installs or upgrades the OpenTelemetry Kube Stack Helm chart, optionally enabling the `host-profiler` collector in the same release;
- installs or upgrades the Datadog Operator (`datadog/datadog-operator` chart) in the `datadog` namespace, with [Fleet Automation](#fleet-automation-optional) enabled when an application key is provided;
- applies the `datadog-agent.yaml` `DatadogAgent` custom resource to the `datadog` namespace, substituting the cluster name and site into it.

To skip the credential prompts on every run, create the `.env` file yourself, see the [installation guide](INSTALL.md#2-provide-the-datadog-credentials). Keep this file out of version control.

Datadog engineers upgrading a cluster set up with an earlier version of this guide: see the [upgrade guide](UPGRADE.md).

## Install with values files

To perform the same installation without the interactive script, create a Kubernetes secret for the Datadog credentials,
install cert-manager, then apply a platform-specific values overlay.

Set the Datadog credentials:

```sh
export DD_API_KEY="<your-datadog-api-key>"
export DD_SITE="datadoghq.com" # Use your Datadog site when different.
```

Create the namespaces and the secret, duplicated into both — `opentelemetry-operator-system` for the OTel collectors' Datadog exporter, `datadog` for the `DatadogAgent` custom resource:

```sh
kubectl create namespace opentelemetry-operator-system \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace datadog \
  --dry-run=client -o yaml | kubectl apply -f -

for NS in opentelemetry-operator-system datadog; do
  kubectl create secret generic datadog-secret \
    --namespace "$NS" \
    --from-literal="api-key=$DD_API_KEY" \
    --from-literal="dd-site=$DD_SITE" \
    --dry-run=client -o yaml | kubectl apply -f -
done
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

For other Kubernetes platforms, start with the manual cluster-name example and replace `my-k8s-cluster` and `production`
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
  --version 0.24.2 \
  --namespace opentelemetry-operator-system \
  --values ./values.yaml \
  --values ./deployment/values.yaml
```

Alternatively, if you want to enable the `host-profiler` collector:

```sh
helm upgrade --install opentelemetry-kube-stack \
  open-telemetry/opentelemetry-kube-stack \
  --version 0.24.2 \
  --namespace opentelemetry-operator-system \
  --set collectors.host-profiler.enabled=true \
  --values ./values.yaml \
  --values ./host-profiler-rbac-values.yaml \
  --values ./deployment/values.yaml
```

On clusters enforcing NetworkPolicy, also apply one of:

```sh
kubectl apply -f ./host-profiler-network-policy.yaml          # any enforcing CNI
# or
kubectl apply -f ./host-profiler-cilium-network-policy.yaml   # Cilium, FQDN-scoped egress
```

Finally, install the [Datadog Operator][dd-operator] and apply the `DatadogAgent` custom resource, substituting the cluster name and site placeholders (`<CLUSTER_NAME>` / `<DD_SITE>`) in `datadog-agent.yaml`. Use the same cluster name as `deployment/values.yaml` above, following the [cluster name constraints](#cluster-name-constraints) (leave `K8S_CLUSTER_NAME` empty on EKS/GKE/AKS, where it's auto-detected instead):

```sh
export K8S_CLUSTER_NAME="my-k8s-cluster" # empty string on EKS/GKE/AKS

helm repo add datadog https://helm.datadoghq.com
helm repo update
helm upgrade --install datadog-operator \
  datadog/datadog-operator \
  --namespace datadog \
  --wait --timeout 5m

# On EKS/GKE/AKS (empty K8S_CLUSTER_NAME), drop the clusterName line so the Agent auto-detects it.
# Likewise, only set the Agent hostname from the Kubernetes node name and the cluster name on non-cloud clusters.
if [[ -n "$K8S_CLUSTER_NAME" ]]; then
  CLUSTER_NAME_SED_ARGS=(-e "s|<CLUSTER_NAME>|$K8S_CLUSTER_NAME|" -e "/<HOSTNAME_FROM_NODE_NAME:/d")
else
  CLUSTER_NAME_SED_ARGS=(-e "/<CLUSTER_NAME>/d" -e "/<HOSTNAME_FROM_NODE_NAME:BEGIN>/,/<HOSTNAME_FROM_NODE_NAME:END>/d")
fi

sed \
  "${CLUSTER_NAME_SED_ARGS[@]}" \
  -e "s|<DD_SITE>|$DD_SITE|" \
  ./datadog-agent.yaml \
  | kubectl apply --namespace datadog -f -
```

## Fleet Automation (optional)

To manage the Datadog Agent from [Fleet Automation][dd-fleet-automation], give the installer a Datadog [application key][dd-app-keys]: it prompts for one during the interactive setup, or add it to `.env` and re-run `./install`:

```sh
export DD_APP_KEY="<your-datadog-application-key>"
```

The installer always gives the Datadog Operator its own API key, site, and cluster name. With an application key, it also stores it as `app-key` in the `datadog` namespace's `datadog-secret`, and installs the Datadog Operator with it and with Remote Configuration enabled (`appKeyExistingSecret`, `remoteConfiguration.enabled=true` and `previewFleetRollouts=true` Helm values). Without an application key, the Operator is installed without Remote Configuration.

- Remote configuration of Agents running on Kubernetes is in Preview: [request access][dd-fleet-k8s-preview] for your Datadog organization.
- The Operator's Remote Configuration requires a cluster name, even on EKS, GKE, and AKS where the Datadog Agent and the OpenTelemetry Collector auto-detect it: the Operator only reads its `clusterName` Helm value. So with an application key, the installer also prompts for the cluster name on EKS, GKE, and AKS, defaulting to the name in the current `kubectl` context, and uses it for the Operator, the Datadog Agent, and the collectors' `k8s.cluster.name`.
- The application key acts with the permissions of the user who created it: prefer a dedicated, scoped application key.

## Host profiler (optional)

The host profiler runs the [Datadog host profiler][dd-host-profiler] (Datadog's own distribution of the [OpenTelemetry eBPF profiler][ebpf-profiler], to which it actively contributes) as a collector DaemonSet on every node and exports profiles to Datadog's OTLP intake.

Enable it by answering `y` to the installer's prompt, or install manually following the [manual installation steps][dd-host-profiler-install]. Unlike the other collectors, its pods need more privileges, which is why it is opt-in.

## Cluster name detection

For EKS, AKS, and GKE, the installer enables the corresponding resource-detection preset in both collectors. The
OpenTelemetry Collector then automatically populates `k8s.cluster.name`.

For other Kubernetes platforms, and on EKS, GKE, and AKS when a cluster name is entered for
[Fleet Automation](#fleet-automation-optional), the installer sets `resourceAttributes.k8s.cluster.name` to the supplied
cluster name, which overrides the detected one.

### Cluster name constraints

The supplied cluster name is used both as the OpenTelemetry `k8s.cluster.name` resource attribute and as the Datadog
Agent's `clusterName`, so it must satisfy the [Datadog Agent's restrictions][dd-cluster-name]. It's made of
dot-separated tokens that:

- only contain lowercase letters, numbers, and hyphens (`-`): no uppercase letters and no underscores (`_`);
- start with a letter;
- end with a letter or a number.

The whole name must be at most 80 characters long. For example, `my-k8s-cluster` and `prod.eu-west-1` are valid;
`my_k8s_cluster`, `My-k8s-cluster`, and `1-cluster` are not.

The Datadog Agent rewrites `_` to `-` and ignores names containing uppercase letters, so an invalid name leaves the
Agent's and the OpenTelemetry Collector's telemetry with different (or missing) cluster names. The installer lowercases
the supplied name and replaces `_` with `-` (with a message if it does); when installing manually, choose a valid name
and use it for both `k8s.cluster.name` and `K8S_CLUSTER_NAME`.

See `examples/` for rendered values and manifests for each deployment type. Regenerate them with `make generate-otel-kube-stack-examples`.

The Datadog Agent is installed as its own step, via the Datadog Operator's `DatadogAgent` custom resource (`datadog-agent.yaml`), in the `datadog` namespace — see [What this deploys](#what-this-deploys) and [Install with values files](#install-with-values-files). See `examples-datadog-agent/` for the manifests the Datadog Operator generates from it, rendered offline with the [datadog-operator][dd-operator]'s `operator-render` CLI (built on the fly from a shallow clone under `tmp/`). Regenerate it with `make generate-datadog-operator-examples`.

Run `make generate-examples` to regenerate both sets of examples in one step.

## Resource allocation

Both collectors default to `500m` CPU / `1Gi` memory limits and `200m` CPU / `500Mi` memory requests. Scale up for large clusters.

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.24.2`
- Collector image `otel/opentelemetry-collector-contrib >= 0.161.0` (pinned in values.yaml under `opentelemetry-operator.manager.collectorImage`)

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/
[dd-operator]: https://github.com/DataDog/datadog-operator
[dd-live-debugger-go]: https://docs.datadoghq.com/tracing/live_debugger/?prog_lang=go
[dd-fleet-automation]: https://docs.datadoghq.com/agent/fleet_automation/
[dd-app-keys]: https://docs.datadoghq.com/account_management/api-app-keys/#application-keys
[dd-cluster-name]: https://docs.datadoghq.com/containers/kubernetes/installation/
[dd-fleet-k8s-preview]: https://www.datadoghq.com/product-preview/configure-agent-kubernetes-operator/

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
