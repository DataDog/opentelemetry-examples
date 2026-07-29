# kube-stack values.yaml

Reference `values.yaml` for the [opentelemetry-kube-stack][chart] Helm chart, configured to send Kubernetes telemetry to Datadog. 

## What this deploys

The `opentelemetry-kube-stack` chart installs the OpenTelemetry Operator and renders one `OpenTelemetryCollector` CR for each entry under `collectors:`. This values file configures two of them:

- **`cluster`** — a single-replica Deployment responsible for cluster-scope telemetry: scraping kube-state-metrics and watching Kubernetes objects.
- **`daemon`** — a DaemonSet running on every node, responsible for node-scope telemetry (host and kubelet metrics) and for terminating the OTLP endpoint that application workloads send traces, logs, and metrics to.

## Prerequisites

- A Kubernetes secret named `datadog-secret` with keys `api-key` (required) and `dd-site` (optional; defaults to `datadoghq.com`).
- [cert-manager][cm] installed in the cluster, for the operator's admission webhook.

## Quickstart

```sh
# 1. Create the Datadog secret in a dedicated namespace
export DD_API_KEY=<YOUR API KEY>
export DD_SITE=datadoghq.com
kubectl create namespace opentelemetry-operator-system
kubectl create secret generic datadog-secret \
  --namespace opentelemetry-operator-system \
  --from-literal="api-key=$DD_API_KEY" \
  --from-literal="dd-site=$DD_SITE"

# 2. Install cert-manager (skip if already installed)
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set crds.enabled=true

# 3. Install the kube-stack chart with this values file
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
helm install opentelemetry-kube-stack open-telemetry/opentelemetry-kube-stack \
  --namespace opentelemetry-operator-system \
  -f values.yaml
```

## Cluster name detection

The `resourceDetection` preset (`eks`, `aks`, `gcp`) populates `k8s.cluster.name` on EKS/AKS/GKE. The `env` detector is off: it would stamp the collector pod's own identity on incoming resources.

For clouds not covered by the built-in detectors, add a `resource/cluster` processor at the start of each pipeline:

```yaml
processors:
  resource/cluster:
    attributes:
      - key: k8s.cluster.name
        value: my-cluster
        action: insert
```

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.20.0`
- Collector image `otel/opentelemetry-collector-contrib >= 0.154.0` (pinned in values.yaml under `opentelemetry-operator.manager.collectorImage`)

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/
