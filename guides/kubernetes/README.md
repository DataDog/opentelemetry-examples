> [!IMPORTANT]  
> OTel Kubernetes Metrics Remapping is in closed preview. If you would like to participate, please reach out to your account team.

</br>

# Kubernetes Integration
This guide documents how to configure OpenTelemetry collectors to support Datadog's Kubernetes Integration.

## Background

### Prerequisite: kube-state-metrics
The Datadog OpenTelemetry Kubernetes Integration relies on [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics). Kube-state-metrics generates a broad set of [metrics](https://github.com/kubernetes/kube-state-metrics/tree/main/docs/metrics) 
that are necessary for full functionality.

### Why are there two collectors?
Many of the metrics collected for the Kubernetes integration report on the _overall Kubernetes cluster_ and are therefore not specific to any _single_ node within cluster. For example, it wouldn't make sense to collect the number of deployments on a cluster on each node - it would result in duplicated metrics. To accomodate this constraint, we deploy two separate collectors:
* A singleton collector deployed as a Kubernetes Deployment focused on collecting cluster level metrics
* A collector deployed as a Kubernetes Daemonset to collect metrics specific to each host

## Collector Components
### Cluster Collector
| Type        | Name                    | Function |
| :---------- | :---------------------- | :------ |
| receiver    | [prometheus][1]         | Scrapes your cluster's kube-state-metrics endpoint |
| receiver    | [k8s_cluster][2]        | Collects additional cluster-level metrics |
| processor   | [cumulativetodelta][4]  | Converts monotonic, cumulative sum and histogram metrics to monotonic, delta metrics |
| processor   | [resource][5]           | Globally sets the `k8s.cluster.name` attribute on resources for tagging purposes |
| processor   | [transform][6]          | Modifies, adds, and deletes resource/datapoint attributes |
| processor   | [groupbyattrs][7]       | Associates kube-state-metrics Pod datapoints with the correct OTel resource by Pod UID |
| processor   | [k8sattributes][8]      | Enriches Kubernetes metrics with additional metadata (ex: annotations/labels) |
| connector   | [count][9]              | Counts Kubernetes metrics to generate `k8s.node.count`, `k8s.job.count`, and `k8s.service.count` |
| exporter    | [datadog][10]           | Ships telemetry to Datadog |

### Node Collector
| Type        | Name                    | Function |
| :---------- | :---------------------- | :------ |
| receiver    | [hostmetrics][11]       | Collects host-level metrics |
| receiver    | [kubeletstats][12]      | Collects node, pod, container, and volume metrics from the Kubelet |
| processor   | [cumulativetodelta][4]  | Converts monotonic, cumulative sum and histogram metrics to monotonic, delta metrics |
| processor   | [deltatorate][13]       | Converts delta sum metrics to rate metrics that are sent as gauges |
| processor   | [resource][5]           | Globally sets the `k8s.cluster.name` attribute on resources for tagging purposes |
| processor   | [transform][6]          | Modifies, adds, and deletes resource/datapoint attributes |
| processor   | [k8sattributes][8]      | Enriches Kubernetes metrics with additional metadata (ex: annotations/labels) |
| processor   | [resourcedetection][14] | Detects environment information from a variety of sources |
| connector   | [datadog/connector][15] | Converts OTel traces into Datadog compatible traces for consumption in Datadog's APM products |
| exporter    | [datadog][10]           | Ships telemetry to Datadog |


## Metadata Updates
Some metrics' metadata must be updated to ensure it is interpreted properly by Datadog. Metadata can be updated by navigating to Metrics > Summary > Edit

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
Unit: byte_in_binary_bytes_family per second
```

## APM Service Infrastructure Instrumentation
Service Infrastructure metrics use a similar instrumention method to Datadog's [Unified Service tagging](https://docs.datadoghq.com/getting_started/tagging/unified_service_tagging/?tab=kubernetes#opentelemetry). 

There are three key OpenTelemetry attributes to ensure APM traces are associated with infrastructure metrics:
1. `service.name`
2. `service.version`
3. `deployment.environment.name`

### Application Changes
An easy way to set APM Service attributes inside of your application is environment variables:

``` yaml
spec:
  containers:
    - name: my-container
      env:
        - name: OTEL_SERVICE_NAME
          value: <SERVICE NAME HERE>
        - name: OTEL_SERVICE_VERSION
          value: <SERVICE VERSION HERE>
        - name: OTEL_ENVIRONMENT
          value: <DEPLOYMENT ENVIRONMENT HERE>
        - name: OTEL_RESOURCE_ATTRIBUTES
          value: >- service.name=$(OTEL_SERVICE_NAME), service.version=$(OTEL_SERVICE_VERSION), deployment.environment.name=$(OTEL_ENVIRONMENT)
```

### Infrastructure changes
Likewise, the same APM Service attributes must be reflected in your Kubernetes manifests as annotations

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  annotations:
    resource.opentelemetry.io/service.name: <SERVICE NAME HERE>
    resource.opentelemetry.io/service.version: <SERVICE VERSION HERE>
    resource.opentelemetry.io/deployment.environment.name: <DEPLOYMENT ENVIRONMENT HERE>
spec:
  template:
    metadata:
      annotations:
        resource.opentelemetry.io/service.name: <SERVICE NAME HERE>
        resource.opentelemetry.io/service.version: <SERVICE VERSION HERE>
        resource.opentelemetry.io/deployment.environment.name: <DEPLOYMENT ENVIRONMENT HERE>
```

## Deployment

<blockquote> NOTE: If you are incorporating the configuration files found in [/configuration](/documentation/kubernetes/configuration/) into your existing OpenTelemetry collector deployment, please be aware that they are specifically written for

* [opentelemetry-collector](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-collector) helm chart 
* OTel image `otel/opentelemetry-collector-contrib` >= `0.130.0`

</blockquote>

### Quickstart
#### Install kube-state-metrics

```sh
# Add the kube-state-metrics helm chart
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install KSM on your cluster
helm install kube-state-metrics prometheus-community/kube-state-metrics
```

#### Create the `DD_API_KEY` Kubernetes Secret
```sh
# Export your API Key
export DD_API_KEY=<YOUR API KEY>

# Create the secret on your cluster
kubectl create secret generic datadog-secret --from-literal api-key=$DD_API_KEY
```

#### Install the Collectors
```bash
# Add the OpenTelemetry helm chart
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```
```bash
# Export your cluster name
export K8S_CLUSTER_NAME=<YOUR CLUSTER NAME HERE>
```

```bash
# Install the Daemonset Collector 
helm install otel-daemon-collector open-telemetry/opentelemetry-collector -f configuration/daemonset-collector.yaml \
  --set image.repository=otel/opentelemetry-collector-contrib \
  --set image.tag=0.130.0 \
  --set-string "config.processors.resource.attributes[0].key=k8s.cluster.name" \
  --set-string "config.processors.resource.attributes[0].value=${K8S_CLUSTER_NAME}"


# Install the Cluster Collector
helm install otel-cluster-collector open-telemetry/opentelemetry-collector -f configuration/cluster-collector.yaml \
  --set image.repository=otel/opentelemetry-collector-contrib \
  --set image.tag=0.130.0 \
  --set-string "config.processors.resource.attributes[0].key=k8s.cluster.name" \
  --set-string "config.processors.resource.attributes[0].value=${K8S_CLUSTER_NAME}"
```

[1]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/prometheusreceiver
[2]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/k8sclusterreceiver
[4]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/cumulativetodeltaprocessor
[5]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourceprocessor
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

