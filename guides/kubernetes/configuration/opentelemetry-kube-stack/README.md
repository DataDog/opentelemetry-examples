# kube-stack values.yaml

Reference values files for the [opentelemetry-kube-stack][chart] Helm chart, configured to send Kubernetes telemetry to Datadog.

## What this deploys

The `opentelemetry-kube-stack` chart installs the OpenTelemetry Operator and renders one `OpenTelemetryCollector` CR for each entry under `collectors:`. These values files configure two of them:

- **`cluster`**: a single-replica Deployment for cluster-scope telemetry. It scrapes kube-state-metrics and watches Kubernetes objects.
- **`daemon`**: a DaemonSet on every node for node-scope telemetry (host and kubelet metrics). It also serves the OTLP endpoint that application workloads send traces, logs, and metrics to.

You can also install the **host profiler** collector, a DaemonSet that runs the OpenTelemetry eBPF profiler on every node and exports profiles to Datadog. See [Host profiler (optional)](#host-profiler-optional).

## Prerequisites

- A Kubernetes secret named `datadog-secret` with keys `api-key` (required) and `dd-site` (optional; defaults to `datadoghq.com`).
- [cert-manager][cm] installed in the cluster, for the operator's admission webhook.
- For the optional host profiler only: Linux nodes with kernel >= 5.10 and Kubernetes >= 1.30.
  The host profiler's `securityContext` uses the container-level `appArmorProfile` field, introduced in Kubernetes 1.30.

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
| `DD_AGENT_ENABLED` | Install the Datadog Agent in the `ddot` flow: `true` or `false`. | `true` |
| `DD_APP_KEY` | Datadog application key. Enables Fleet Automation for the Datadog Agent. Requires `K8S_CLUSTER_NAME`. | Unset |
| `DD_HOST_PROFILER_ENABLED` | Enable the eBPF host profiler: `true` or `false`. See [Host profiler (optional)](#host-profiler-optional). | `false` |
| `DD_HOST_PROFILER_NETWORK_POLICY` | Host profiler egress NetworkPolicy: `none`, `standard`, or `cilium`. | `none` |

```sh
export DD_API_KEY="<your-datadog-api-key>"
export DD_INSTALL_FLOW=upstream
export K8S_CLUSTER_TYPE=eks
./install
```

You can also copy `.env.example` to `.env`, fill it in, and load it before you run the installer:

```sh
cp .env.example .env
# Edit .env
source .env
./install
```

The installer:

- creates the `opentelemetry-operator-system` namespace and the `datadog-secret` secret;
- installs cert-manager when needed;
- installs or upgrades the OpenTelemetry Kube Stack Helm chart for the selected flow, with the resource-detection preset for EKS, GKE, or AKS and, if enabled, the host profiler; and
- for `ddot`, unless `DD_AGENT_ENABLED=false`, installs or upgrades the Datadog Agent.

## Install with values files

Set the Datadog credentials:

```sh
export DD_API_KEY="<your-datadog-api-key>"
export DD_SITE="datadoghq.com" # Use your Datadog site when different.
```

Create the namespace and the secret that the collectors read:

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

Copy the example for your cluster platform to `deployment/values.yaml`, then set its deployment environment.
The EKS, GKE, and AKS examples enable the matching resource detector, which sets `k8s.cluster.name` automatically:

```sh
mkdir -p deployment

# Choose one:
cp examples/eks-deployment/values.yaml deployment/values.yaml
cp examples/gcp-deployment/values.yaml deployment/values.yaml
cp examples/aks-deployment/values.yaml deployment/values.yaml
```

For other Kubernetes platforms, copy the manual cluster-name example, then replace `my_k8s_cluster` and `production`
with your cluster name and deployment environment. The collectors still read `DD_SITE` from `datadog-secret`.

```sh
mkdir -p deployment
cp examples/manually-set-k8s-cluster-name/values.yaml deployment/values.yaml
```

`deployment/values.yaml` is an overlay for the base `values-otlp-http.yaml` in this directory. Add any deployment-specific
configuration to it, then install or upgrade the chart with both files:

```sh
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
helm upgrade --install opentelemetry-kube-stack \
  open-telemetry/opentelemetry-kube-stack \
  --version 0.26.1 \
  --namespace opentelemetry-operator-system \
  --values ./values-otlp-http.yaml \
  --values ./deployment/values.yaml
```

## Host profiler (optional)

The host profiler collector runs the [Datadog host profiler][dd-host-profiler], Datadog's distribution of the
[OpenTelemetry eBPF profiler][ebpf-profiler], as a DaemonSet on every node. It exports profiles to Datadog's OTLP intake.

Its pods need more privileges than the other collectors, so it is opt-in. To enable it, set
`DD_HOST_PROFILER_ENABLED=true` before you run the installer.

On clusters that enforce NetworkPolicy, set `DD_HOST_PROFILER_NETWORK_POLICY` to restrict the host profiler's egress:
`standard` works with any CNI that enforces NetworkPolicy, and `cilium` scopes egress by FQDN.

## Cluster name detection

For EKS, AKS, and GKE, the installer enables the matching resource-detection preset in both collectors. The
OpenTelemetry Collector then sets `k8s.cluster.name` automatically, unless `K8S_CLUSTER_NAME` is set.

For other Kubernetes platforms, the installer sets `resourceAttributes.k8s.cluster.name` to `K8S_CLUSTER_NAME` if that
variable is set.

See `examples/` for rendered values and manifests for each deployment type.

## Resource allocation

Both collectors default to `500m` CPU / `1Gi` memory limits and `200m` CPU / `500Mi` memory requests. Scale up for large clusters.

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.26.1`
- Collector image `otel/opentelemetry-collector-contrib >= 0.163.1` (set in the values files under `opentelemetry-operator.manager.collectorImage`)

## Troubleshooting

For OpenTelemetry Operator metrics troubleshooting, see [TROUBLESHOOTING.md](ddot/TROUBLESHOOTING.md).

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/
[dd-host-profiler]: https://github.com/DataDog/datadog-agent/tree/main/cmd/host-profiler
[ebpf-profiler]: https://github.com/open-telemetry/opentelemetry-ebpf-profiler
