# Collector Configurations

This directory contains example configuration files to deploy the OpenTelemetry Collector and have it export telemetry to Datadog.

Which configuration file to use depends on four factors:

1. Deployment method
    - Manual
    - Collector Helm chart
    - OTel Demo Helm chart
2. Deployment pattern:
    - [Agent deployment pattern](https://opentelemetry.io/docs/collector/deploy/agent/)
3. Execution environment:
    - Uncontainerized (bare-metal or VM)
    - Unorchestrated container (Docker)
    - Kubernetes
4. Cloud provider:
    - Non-cloud (on-premises)
    - Amazon Web Services (EC2, EKS, etc.)
    - Google Cloud Provider (GCE, etc.)
    - Microsoft Azure

Setups not listed above are not necessarily unsupported, and we may add recommended configurations for them in the future.

Make sure to read the comments in the configuration file you decide to use, as many of them have external dependencies (e.g. required secrets or environment variables), or fields that must be filled in before use (e.g. DD_SITE).

## Raw Collector configurations (manual deployment)

These files are the raw YAML configuration we recommend passing to a Collector using the `--config` flag.

Example:
```sh
DD_SITE=datadoghq.com DD_API_KEY='insertyourapikey' ./otelcol-contrib --config ./otelcol-host.yaml --feature-gates connector.spanmetrics.includeCollectorInstanceID
```

### Agent deployment pattern

Can be a Daemonset deployment (Kubernetes) or a manual (containerized or not) Agent deployment.

Files:
- `otelcol-agent.yaml`: Uncontainerized Agent in a non-cloud environment
- `otelcol-agent-gce.yaml`: Uncontainerized Agent in a GCE environment
- `otelcol-agent-ec2.yaml`: Uncontainerized Agent in an EC2 environment
- `otelcol-agent-azure.yaml`: Uncontainerized Agent in an Azure VM environment
- `otelcol-agent-container.yaml`: Containerized Agent in a non-cloud environment
- `otelcol-daemonset.yaml`: Kubernetes Daemonset in a non-cloud environment
- `otelcol-daemonset-eks.yaml`: Kubernetes Daemonset in an EKS environment

## Deployment using the Collector Helm chart

A simple way to deploy the Collector in Kubernetes is to use [the official Helm chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-collector).

The following files are `values.yaml` files to be passed as a `--values` flag when deploying the Collector Helm chart.
They are generated from the above raw Collector configurations and automate setting up some of the necessary feature gates / mounts.

Example:
```sh
kubectl create secret generic datadog-secrets --from-literal=api-key='insertyourapikey'
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm install otelcol open-telemetry/opentelemetry-collector --values ./helm-daemonset.yaml
```

Files:
- `helm-daemonset.yaml`: Kubernetes Daemonset in a non-cloud environment
- `helm-daemonset-eks.yaml`: Kubernetes Daemonset in an EKS environment

## Deploying the OpenTelemetry Demo

The [OpenTelemetry Demo](https://github.com/open-telemetry/opentelemetry-demo) is a community-built example application which serves to demonstrate the various features of OpenTelemetry. Like the Collector itself, it can be deployed in Kubernetes using [the official Helm chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-demo).

The following files are `values.yaml` files to be passed as a `--values` flag when deploying the Demo Helm chart.
These will deploy a mostly vanilla version of the OpenTelemetry Demo, whose telemetry is exported via a Daemonset Collector using one of the above recommended configurations.

Example:
```sh
kubectl create secret generic datadog-secrets --from-literal=api-key='insertyourapikey'
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm install otel-demo open-telemetry/opentelemetry-demo --values ./otel-demo.yaml
```

Files:
- `otel-demo.yaml`: Kubernetes Daemonset in a non-cloud environment
- `otel-demo-eks.yaml`: Kubernetes Daemonset in an EKS environment

The `otel-demo-testing-*.yaml` files are for internal testing purposes and are not recommended for general use.
