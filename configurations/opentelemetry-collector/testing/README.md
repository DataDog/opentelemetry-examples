# Configurations for internal testing

This directory contains Collector configurations for internal testing purposes.

> [!WARNING]  
> These files are not recommended for use by Datadog users,
> and are only meant for testing and comparison by Datadog employees.

## Raw Collector configurations for manual deployment

Files:
- `agent-datadog.yaml`: Uncontainerized Agent in a non-cloud environment using the Datadog exporter
- `daemonset-datadog.yaml`: Kubernetes Daemonset in a non-cloud environment using the Datadog exporter

These files are for use in the `pipeline-test` internal testing tool.

## Values files for the OpenTelemetry Demo Helm chart

The [OpenTelemetry Demo](https://github.com/open-telemetry/opentelemetry-demo) is a community-built example application which serves to demonstrate the various features of OpenTelemetry. Like the Collector itself, it can be deployed in Kubernetes using [the official Helm chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-demo).

The following files are `values.yaml` files to be passed as a `--values` flag when deploying the Demo Helm chart.
These will deploy a mostly vanilla version of the OpenTelemetry Demo, whose telemetry is exported via a Daemonset Collector using one of the above recommended configurations.

Example:
```sh
kubectl create secret generic datadog-secrets --from-literal=api-key='insertyourapikey'
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm install otel-demo open-telemetry/opentelemetry-demo --values ./testing/otel-demo.yaml
```

Files:
- `otel-demo.yaml`: Kubernetes Daemonset in a non-cloud environment
- `otel-demo-eks.yaml`: Kubernetes Daemonset in an EKS environment
- `otel-demo-datadog.yaml`: Kubernetes Daemonset in a non-cloud environment using the Datadog exporter
- `otel-demo-datadog-eks.yaml`: Kubernetes Daemonset in an EKS environment using the Datadog exporter

