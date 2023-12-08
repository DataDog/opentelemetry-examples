# Collector Configurations

This folder contains examples of OpenTelemetry Collector configuration files that we support for exporting to Datadog.
`minimal-config.yaml` and `multiple-pipelines.yaml` are both pipeline configurations with the following flow of data:

[OTLP Receiver](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/otlpreceiver/README.md) → [Batch Processor](https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/batchprocessor/README.md) → [Datadog Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/datadogexporter)

## Configuration Files
These files define [Collector configurations](https://opentelemetry.io/docs/collector/configuration/) that can be used when building or running the Collector.

### `minimal-config.yaml`
This is the minimal pipeline configuration we support for exporting to Datadog. This file defines two pipelines, one for metrics and one for traces.

### `multiple-pipelines.yaml`
This is a more complicated configuration: it does dual-shipping to EU and US and has two metrics pipelines.

### `sidecar-config.yaml`
This example defines a sidecar Collector, that exports data to another Collector.

## Deployment Files
These files define specifications and Collector configurations that can be used in the process of deploying to Kubernetes.

### `minimal-helm-values.yaml`
This is an example of a minimal [helm values file](https://helm.sh/docs/chart_template_guide/values_files/) that we support for exporting to Datadog. This file defines a similar pipeline to `minimal-config.yaml` with the addition of the [Resource Detection Processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/processor/resourcedetectionprocessor/README.md) and environment variables.

### `daemonset.yaml`
This file can be used for deploying a basic OpenTelemetry Collector for sending OpenTelemetry traces to Datadog. It is configured with the Datadog Exporter and the OTLP receiver.

## Installing on Kubernetes
Switch to your desired context and add the helm charts repo:
```
kubectx <desired context>
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
```

Create a secret with your datadog api key (alternatively you may update the secret in values.yaml or --set it when installing):
```
kubectl create secret generic datadog-secrets --from-literal api-key=<YOUR_KEY_HERE>
```

Install the helm chart:
```
helm install my-opentelemetry-collector open-telemetry/opentelemetry-collector -f <YOUR_VALUES_FILE>.yaml
```

See OpenTelemetry [docs](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-collector) for more context.