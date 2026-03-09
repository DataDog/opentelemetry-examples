# Rolldice HTTP Metrics Helm Chart

This Helm chart deploys the rolldice application with HTTP metrics collection using OpenTelemetry on Kubernetes.

![Services Architecture](../services-diagram.png)

## Prerequisites

- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html) with the `sso-dev-opentelemetry-account-admin` profile configured
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Helm 3.8+](https://helm.sh/docs/intro/install/)
- Docker (only if building and pushing new images)

## Cluster Access

All developers access the cluster via a stable IAM role (`otel-demo-eks-admin`) that any `account-admin` SSO user can assume. No manual access entry registration is required.

### 1. Log in to AWS SSO

```bash
aws sso login --profile sso-dev-opentelemetry-account-admin
```

### 2. Set up kubectl

The kubeconfig is committed to this repo. Point kubectl at it:

```bash
export KUBECONFIG=~/.kube/config:$(git rev-parse --show-toplevel)/apps/rolldice-http-metrics/kubeconfig.yaml
```

Add this line to your `~/.zshrc` or `~/.bashrc` to make it permanent.

Verify access:

```bash
kubectl get nodes
```

## Deployment

### 1. Create the Datadog API Key Secret

```bash
kubectl create secret generic datadog-secret \
  --from-literal=api-key=your-datadog-api-key
```

This secret is referenced by both the OTel collector and the Datadog agent. It must exist before installing the chart.

### 2. Install the Chart

For EKS deployment using the pre-configured ECR images:

```bash
helm install rolldice ./helm-chart -f values-deploy.yaml
```

Verify all pods come up:

```bash
kubectl get pods -l app.kubernetes.io/instance=rolldice
```

All pods should reach `1/1 Running` within ~2 minutes.

### 3. Test the Application

```bash
# Port forward to the game controller
kubectl port-forward svc/rolldice-game-controller 5002:5002

# Play a game
curl -X POST http://localhost:5002/play_game \
  -H "Content-Type: application/json" \
  -d '{"player": "testuser"}'
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.useExponentialHistograms` | Enable exponential histograms for HTTP metrics | `false` |
| `global.useOldConventions` | Use old OTel HTTP semantic conventions | `false` |
| `global.useSpanMetricsConnector` | Use span metrics connector instead of routing connector | `false` |
| `otelCollector.datadog.apiKeySecret.name` | Name of the secret containing the Datadog API key | `""` |
| `otelCollector.datadog.apiKeySecret.key` | Key within the secret | `"api-key"` |
| `otelCollector.datadog.site` | Datadog site | `"datadoghq.com"` |
| `image.registry` | Container registry prefix | `""` |
| `gameController.replicaCount` | Number of game controller replicas | `1` |
| `rolling.replicaCount` | Number of rolling service replicas | `1` |
| `scoring.replicaCount` | Number of scoring service replicas | `1` |

### Values Files

| File | Purpose |
|------|---------|
| `values.yaml` | Defaults |
| `values-deploy.yaml` | EKS deployment with ECR images and Datadog agent enabled |
| `values-exponential.yaml` | Exponential histogram variant |
| `values-old-conventions.yaml` | Old OTel HTTP semantic conventions |
| `values-spanmetricsconnector.yaml` | Span metrics connector variant |

## Scaling

```bash
# Scale a deployment
kubectl scale deployment rolldice-game-controller --replicas=3

# Or via Helm
helm upgrade rolldice ./helm-chart -f values-deploy.yaml \
  --set gameController.replicaCount=3
```

## Troubleshooting

### Check pod status
```bash
kubectl get pods -l app.kubernetes.io/instance=rolldice
```

### View logs
```bash
# Game controller
kubectl logs -l app.kubernetes.io/name=game-controller

# OTel collector
kubectl logs -l app.kubernetes.io/name=otelcol

# Load generator
kubectl logs -l app.kubernetes.io/name=load-generator
```

### Check OTel collector config
```bash
kubectl get configmap rolldice-otel-config -o yaml
```

### Check services
```bash
kubectl get svc -l app.kubernetes.io/instance=rolldice
```

## Cleanup

```bash
helm uninstall rolldice
kubectl delete secret datadog-secret
```
