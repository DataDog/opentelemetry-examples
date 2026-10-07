# opentelemetry-kube-stack

Reference configuration for the [opentelemetry-kube-stack][chart] Helm chart to use with Datadog.

## What this deploys

The `opentelemetry-kube-stack` chart installs the OpenTelemetry Operator and renders one `OpenTelemetryCollector` CR for each entry under `collectors:`. This configuration defines two collectors:

- **`cluster`**: a single-replica Deployment for cluster-scope telemetry. It scrapes kube-state-metrics and watches Kubernetes objects.
- **`daemon`**: a DaemonSet for node-scope telemetry (host and kubelet metrics). It also receives OTLP traces, logs, and metrics from application workloads.

You can also install the Datadog host profiler collector. See [Host profiler (optional)](#host-profiler-optional).

There are two installation flows:

1. The `upstream` flow uses the OpenTelemetry SDKs and the OTLP/HTTP exporters to send data via the [Datadog OTLP Intake Endpoints][otlp-endpoints].
2. The `ddot` flow uses the DDOT SDKs, the DDOT Collector and optionally the Datadog Agent to talk with Datadog. It gives access to Datadog Agent only features not available in upstream OpenTelemetry instrumentation.

## Prerequisites

- A Kubernetes cluster.
- `kubectl` and `helm` configured for the cluster, with permission to install cluster-scoped resources.
- For the optional host profiler only: Linux nodes with kernel >= 5.10 and Kubernetes >= 1.30.

## Quickstart

Configure the installer with environment variables, then run it from this directory:

| Variable | Description | Default |
|---|---|---|
| `DD_API_KEY` | Datadog API key. | None (required) |
| `DD_INSTALL_FLOW` | Installation flow: `upstream` or `ddot` (see above). | None (required) |
| `DD_SITE` | [Datadog site][dd-site]. | `datadoghq.com` |
| `K8S_CLUSTER_TYPE` | Kubernetes platform: `eks` (EKS), `gcp` (GKE), `aks` (AKS), or `other`. | `other` |
| `K8S_CLUSTER_NAME` | Kubernetes cluster name. For `eks`, `gcp`, and `aks`, overrides the auto-detected name. | Auto-detected for `eks`, `gcp`, and `aks`; unset otherwise |
| `DEPLOYMENT_ENVIRONMENT_NAME` | Value of the `deployment.environment.name` resource attribute. | `production` |
| `DD_HOST_PROFILER_ENABLED` | Enable the eBPF host profiler: `true` or `false`. See [Host profiler (optional)](#host-profiler-optional). | `false` |
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

The installer:

- creates the `opentelemetry-operator-system` namespace and the `datadog-secret` secret;
- installs cert-manager if needed;
- installs or upgrades the opentelemetry-kube-stack chart; and
- for `ddot`, installs or upgrades the Datadog Agent chart.

## Host profiler (optional)

The host profiler collector runs the [Datadog host profiler][dd-host-profiler], Datadog's distribution of the [OpenTelemetry eBPF profiler][ebpf-profiler], as a DaemonSet and exports profiles to Datadog.

The host profiler is opt-in because its pods need more privileges than the other collectors. To enable it, set `DD_HOST_PROFILER_ENABLED=true`. To restrict its egress on clusters that enforce NetworkPolicy, set `DD_HOST_PROFILER_NETWORK_POLICY` to `standard`, or to `cilium` for FQDN-scoped egress on Cilium.

## Resource allocation

Both collectors default to `500m` CPU / `1Gi` memory limits and `200m` CPU / `500Mi` memory requests. Scale up for large clusters.

## Chart version

Verified against:

- `opentelemetry-kube-stack` chart `>= 0.23.0`
- Collector image `otel/opentelemetry-collector-contrib >= 0.161.0` (pinned under `opentelemetry-operator.manager.collectorImage`)

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[dd-host-profiler]: https://github.com/DataDog/datadog-agent/tree/main/cmd/host-profiler
[dd-site]: https://docs.datadoghq.com/getting_started/site/
[ebpf-profiler]: https://github.com/open-telemetry/opentelemetry-ebpf-profiler
[otlp-endpoints]: https://docs.datadoghq.com/opentelemetry/setup/otlp_ingest/
