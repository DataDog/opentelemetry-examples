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
# 1. Create the Datadog secret
export DD_API_KEY=<YOUR API KEY>
export DD_SITE=datadoghq.com
kubectl create secret generic datadog-secret \
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
helm install otel-k8s open-telemetry/opentelemetry-kube-stack -f values.yaml
```

## Cluster name detection

Both collectors run `resourcedetection` with `k8s_api, ec2, eks, aks, gcp, env` detectors to populate `k8s.cluster.name` and `k8s.cluster.uid`. The `eks`, `aks`, and `gcp` detectors handle EKS/AKS/GKE automatically.

If your cloud provider isn't supported, set the chart's top-level `clusterName` to your cluster name. The chart injects it into `OTEL_RESOURCE_ATTRIBUTES` and the `env` detector picks it up.

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.19.1`
- Collector image `otel/opentelemetry-collector-contrib >= 0.154.0` (pinned in values.yaml under `opentelemetry-operator.manager.collectorImage`)

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/
