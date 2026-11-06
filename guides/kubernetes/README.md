# Kubernetes Integration
This guide shows how to configure OpenTelemetry Collectors for the Datadog Kubernetes integration.

## Background

### Prerequisite: kube-state-metrics
For full functionality, the integration requires the [metrics](https://github.com/kubernetes/kube-state-metrics/tree/main/docs/metrics) that [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) generates.

### Why are there two collectors?
Cluster-level metrics, such as the number of Deployments, would be duplicated if every node collected them. This setup follows the [OpenTelemetry Kubernetes pattern](https://opentelemetry.io/docs/platforms/kubernetes/getting-started/) and deploys two collectors:
* A single-replica Deployment that collects cluster-level metrics
* A DaemonSet that collects metrics for each host

## Collector Components
### Cluster Collector
| Type        | Name                    | Function |
| :---------- | :---------------------- | :------ |
| receiver    | [prometheus][1]         | Scrapes your cluster's kube-state-metrics endpoint |
| receiver    | [k8s_cluster][2]        | Collects additional cluster-level metrics |
| receiver    | [k8sobjects][3]         | Collects Kubernetes objects for the Kubernetes Explorer |
| processor   | [cumulativetodelta][4]  | Converts monotonic, cumulative sum and histogram metrics to monotonic, delta metrics |
| processor   | [resourcedetection][14] | Automatically detects [`k8s.cluster.name`][16] (EKS/AKS/GKE) and `k8s.cluster.uid` (kubeadm) |
| processor   | [transform][6]          | Modifies, adds, and deletes resource/datapoint attributes |
| processor   | [groupbyattrs][7]       | Associates kube-state-metrics Pod datapoints with the correct OTel resource by Pod UID |
| processor   | [k8s_attributes][8]     | Enriches Kubernetes metrics with additional metadata (for example, annotations and labels) |
| connector   | [count][9]              | Counts Kubernetes metrics to generate `k8s.node.count`, `k8s.job.count`, `k8s.service.count`, `k8s.deployment.count`, `k8s.pod.count`, `k8s.namespace.count`, `k8s.statefulset.count`, `k8s.daemonset.count`, and `k8s.cronjob.count` |
| exporter    | [datadog][10]           | Ships telemetry to Datadog |

### Node Collector
| Type        | Name                    | Function |
| :---------- | :---------------------- | :------ |
| receiver    | [hostmetrics][11]       | Collects host-level metrics |
| receiver    | [kubeletstats][12]      | Collects node, pod, container, and volume metrics from the Kubelet |
| processor   | [cumulativetodelta][4]  | Converts monotonic, cumulative sum and histogram metrics to monotonic, delta metrics |
| processor   | [deltatorate][13]       | Converts delta sum metrics to rate metrics that are sent as gauges |
| processor   | [resourcedetection][14] | Automatically detects [`k8s.cluster.name`][16] (EKS/AKS/GKE) and `k8s.cluster.uid` (kubeadm) |
| processor   | [k8s_attributes][8]     | Enriches Kubernetes metrics with additional metadata (for example, annotations and labels) |
| connector   | [datadog/connector][15] | Converts OTel traces into Datadog-compatible traces for Datadog APM |
| exporter    | [datadog][10]           | Ships telemetry to Datadog |


## Metadata Updates
Update the metadata of the following metrics so Datadog interprets them correctly. To edit metadata, go to **Metrics > Summary > Edit**. See [Metrics Summary](https://docs.datadoghq.com/metrics/summary/).

### k8s.pod.cpu.usage
```
Metric Type: Gauge
Unit: core
```

### k8s.pod.memory.usage
```
Metric Type: Gauge
Unit: byte_in_binary_bytes_family
```

### k8s.pod.network.io
```
Metric Type: Gauge
Unit: byte_in_binary_bytes_family per second
```

### k8s.pod.network.errors
```
Metric Type: Gauge
Unit: error per second
```

### kube_pod_container_status_restarts_total
```
Metric Type: Gauge
```

## APM Service Infrastructure Instrumentation
Service Infrastructure metrics use an instrumentation method similar to Datadog's [Unified Service tagging](https://docs.datadoghq.com/getting_started/tagging/unified_service_tagging/?tab=kubernetes#opentelemetry).

To associate APM traces with infrastructure metrics, set these OpenTelemetry attributes:
1. `service.name`
2. `service.version`
3. `deployment.environment.name`

Set them with the [`OTEL_RESOURCE_ATTRIBUTES`](https://opentelemetry.io/docs/languages/sdk-configuration/general/#otel_resource_attributes)
environment variable, or with [`resource.opentelemetry.io/` pod annotations](https://opentelemetry.io/docs/platforms/kubernetes/operator/automatic/#configure-resource-attributes-with-annotations).
The Collector configurations in this guide enable `otel_annotations` on the `k8s_attributes` processor, so the
annotations also apply to the pod's infrastructure telemetry.

## Deployment

> [!NOTE]
> [`cluster-collector.yaml`](configuration/cluster-collector.yaml) and
> [`daemonset-collector.yaml`](configuration/daemonset-collector.yaml) are written for:
>
> * [opentelemetry-collector](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-collector) Helm chart >= `0.156.2`
> * OTel image `otel/opentelemetry-collector-contrib` >= `0.154.0`

### Quickstart
#### Install kube-state-metrics

```sh
# Add the kube-state-metrics helm chart
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install kube-state-metrics on your cluster
helm install kube-state-metrics prometheus-community/kube-state-metrics
```

#### Create the Kubernetes Secret for `DD_API_KEY` and `DD_SITE`
```sh
# Export your API Key
export DD_API_KEY=<YOUR API KEY>

# Export your Datadog site: datadoghq.com, us3.datadoghq.com, datadoghq.eu...
export DD_SITE=datadoghq.com

# Create the secret on your cluster
kubectl create secret generic datadog-secret --from-literal="api-key=$DD_API_KEY" --from-literal="dd-site=${DD_SITE:-datadoghq.com}"
```

#### Install the Collectors
```bash
# Add the OpenTelemetry helm chart
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```
```bash
# Install the Daemonset Collector
helm install otel-daemon-collector open-telemetry/opentelemetry-collector -f configuration/daemonset-collector.yaml \
  --set image.repository=otel/opentelemetry-collector-contrib \
  --set image.tag=0.154.0

# Install the Cluster Collector
helm install otel-cluster-collector open-telemetry/opentelemetry-collector -f configuration/cluster-collector.yaml \
  --set image.repository=otel/opentelemetry-collector-contrib \
  --set image.tag=0.154.0
```

> [!NOTE]
> The `resourcedetection` processor detects the cluster name on EKS, AKS, and GKE. See the [supported providers][16]. On other providers, edit both configuration files:
>
> 1. Uncomment the `resource/add-cluster-name` processor block and set your cluster name.
> 2. Add `resource/add-cluster-name` to the `processors` list in all pipelines.

[1]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/prometheusreceiver
[2]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/k8sclusterreceiver
[3]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/k8sobjectsreceiver
[4]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/cumulativetodeltaprocessor
[6]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor
[7]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/groupbyattrsprocessor
[8]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/k8sattributesprocessor
[9]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/countconnector
[10]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/datadogexporter
[11]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/hostmetricsreceiver
[12]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/kubeletstatsreceiver
[13]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/deltatorateprocessor
[14]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor
[15]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/datadogconnector
[16]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.153.0/processor/resourcedetectionprocessor#supported-detectors

