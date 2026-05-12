# Collector Configurations

This directory contains example configuration files to deploy the OpenTelemetry Collector and have it export telemetry to Datadog.

These files are generated from a set of templates using the Go tool in the `generator` directory.

Which configuration file to use depends on four factors:

1. Deployment method
    - Manual
    - Collector Helm chart
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

## Raw Collector configurations for manual deployment

These files are the raw YAML configuration we recommend passing to a Collector using the `--config` flag.

Example:
```sh
DD_SITE=datadoghq.com DD_API_KEY='insertyourapikey' ./otelcol-contrib --config ./agent.yaml --feature-gates connector.spanmetrics.includeCollectorInstanceID
```

### Agent deployment pattern

Can be a Daemonset deployment (Kubernetes) or a manual (containerized or not) Agent deployment.

Files:
- `agent.yaml`: Uncontainerized Agent in a non-cloud environment
- `agent-azure.yaml`: Uncontainerized Agent in an Azure VM environment
- `agent-ec2.yaml`: Uncontainerized Agent in an EC2 environment
- `agent-gce.yaml`: Uncontainerized Agent in a GCE environment
- `agent-container.yaml`: Containerized Agent in a non-cloud environment
- `daemonset.yaml`: Kubernetes Daemonset in a non-cloud environment
- `daemonset-eks.yaml`: Kubernetes Daemonset in an EKS environment

## Values files for the Collector Helm chart

A simple way to deploy the Collector in Kubernetes is to use [the official Helm chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-collector).

The following files are `values.yaml` files to be passed as a `--values` flag when deploying the Collector Helm chart.
They are generated from the above raw Collector configurations and automate setting up some of the necessary feature gates / mounts.

Example:
```sh
kubectl create secret generic datadog-secrets --from-literal=api-key='insertyourapikey'
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm install otelcol open-telemetry/opentelemetry-collector --values ./helm-values/daemonset.yaml
```

Files:
- `helm-values/daemonset.yaml`: Kubernetes Daemonset in a non-cloud environment
- `helm-values/daemonset-eks.yaml`: Kubernetes Daemonset in an EKS environment
