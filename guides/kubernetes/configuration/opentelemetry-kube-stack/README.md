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

## Cluster name detection

For EKS, AKS, and GKE, the installer enables the corresponding resource-detection preset in both collectors. The OpenTelemetry Collector then automatically populates `k8s.cluster.name`.

For other Kubernetes platforms, the installer passes the supplied cluster name through `clusterName` and `defaultCRConfig.env[2].value`. The `transform/insert_k8s_cluster_name` processor adds it only when the resource does not already have a `k8s.cluster.name` attribute.

See `examples/` for rendered values and manifests for each deployment type.

## Resource allocation

Both collectors default to `500m` CPU / `1Gi` memory limits and `200m` CPU / `500Mi` memory requests. Scale up for large clusters.

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.20.1`
- Collector image `otel/opentelemetry-collector-contrib >= 0.154.0` (pinned in values.yaml under `opentelemetry-operator.manager.collectorImage`)

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/
