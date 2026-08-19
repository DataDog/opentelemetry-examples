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

Run the installer from this directory:

```sh
./install
```

The installer prompts for your Datadog API key and site (the site defaults to `datadoghq.com`), Kubernetes platform, and deployment environment. For EKS, GKE, and AKS, it enables the matching resource-detection preset. For other platforms, it prompts for the Kubernetes cluster name.

It then:

- creates the `opentelemetry-operator-system` namespace and the `datadog-secret` secret;
- installs cert-manager when needed; and
- installs or upgrades the OpenTelemetry Kube Stack Helm chart.

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
  --set crds.enabled=true
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

For other Kubernetes platforms, start with the manual cluster-name example and replace `my_k8s_cluster`, and
`production` with the cluster name, and deployment environment:

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
  --version 0.20.2 \
  --namespace opentelemetry-operator-system \
  --values ./values.yaml \
  --values ./deployment/values.yaml
```

## Cluster name detection

For EKS, AKS, and GKE, the installer enables the corresponding resource-detection preset in both collectors. The
OpenTelemetry Collector then automatically populates `k8s.cluster.name`.

For other Kubernetes platforms, the installer passes the supplied cluster name through `clusterName` and
`defaultCRConfig.env[2].value`. The `transform/insert_k8s_cluster_name` processor adds it only when the resource does
not already have a `k8s.cluster.name` attribute.

See `examples/` for rendered values and manifests for each deployment type.

## Resource allocation

Both collectors default to `500m` CPU / `1Gi` memory limits and `200m` CPU / `500Mi` memory requests. Scale up for large clusters.

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.20.2`
- Collector image `otel/opentelemetry-collector-contrib >= 0.154.0` (pinned in values.yaml under `opentelemetry-operator.manager.collectorImage`)

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/
