# Rolldice HTTP Metrics Demo

This demo application demonstrates OpenTelemetry instrumentation for HTTP metrics across multiple languages. It consists of a Java Spring Boot game controller that orchestrates two Python Flask services to simulate a dice rolling game.

## Architecture

![Services Architecture](./services-diagram.png)

The application consists of:
- **Game Controller** (Java Spring Boot) - Orchestrates game requests
- **Rolling Service** (Python Flask) - Simulates dice rolling
- **Scoring Service** (Python Flask) - Tracks player scores
- **Load Generator** (Python) - Generates realistic traffic patterns
- **OpenTelemetry Collector** - Collects and exports telemetry to Datadog

## Quick Start with Docker Compose

### Prerequisites
- Docker and Docker Compose installed
- Datadog API key

### Running the Demo

1. Set your Datadog API key:
```bash
export DD_API_KEY=your-datadog-api-key
```

2. (Optional) Set your Datadog site (defaults to datadoghq.com):
```bash
export DD_SITE=datadoghq.com  # or datadoghq.eu for EU
```

3. Start all services:
```bash
docker-compose up
```

4. Test the application:
```bash
# Successful game
curl -X POST http://localhost:5002/play_game \
     -H "Content-Type: application/json" \
     -d '{"player": "John Doe"}'

# Error case (missing player)
curl -X POST http://localhost:5002/play_game \
     -H "Content-Type: application/json" \
     -d '{}'
```

5. View traces and metrics in your Datadog account at `https://app.datadoghq.com`

## Kubernetes Deployment

For production deployments using Kubernetes and Helm, see the [Helm Chart documentation](./helm-chart/README.md).

### AWS Authentication

This demo is deployed to the `dev-opentelemetry` AWS account. Access is granted to any user with the `account-admin` SSO permission set via a stable IAM role — no manual cluster registration required.

**1. Log in to AWS SSO:**

```bash
aws sso login --profile sso-dev-opentelemetry-account-admin
```

**2. Configure kubectl:**

The kubeconfig is committed to this repo. Merge it into your local kubectl config:

```bash
export KUBECONFIG=~/.kube/config:$(git rev-parse --show-toplevel)/apps/rolldice-http-metrics/kubeconfig.yaml
```

Add this to your `~/.zshrc` or `~/.bashrc` to make it permanent.

**3. Verify access:**

```bash
kubectl get nodes
```

Your SSO session lasts 8 hours. Re-run `aws sso login` when it expires — no other steps needed.

### Quick Helm Install

1. Create Datadog API key secret:
```bash
kubectl create secret generic datadog-secret \
  --from-literal=api-key=your-datadog-api-key
```

2. Install the chart:
```bash
helm install rolldice ./helm-chart \
  --set otelCollector.datadog.apiKeySecret.name="datadog-secret"
```

## Configuration

### Environment Variables

The OpenTelemetry Collector supports the following environment variables:

- `DD_API_KEY` - Your Datadog API key (required)
- `DD_SITE` - Datadog site URL (default: `datadoghq.com`)

### OpenTelemetry Features

This demo showcases:
- HTTP client and server metrics with semantic conventions
- Exponential histogram support (configurable)
- Delta temporality for counters and histograms
- Span metrics connector for generating RED metrics from traces
- Custom metrics (games played counter)

## Local Development

### Running Individual Services

Each service can be run locally for development. See the README in each service directory:
- [Game Controller](./game_controller/README.md)
- [Rolling Service](./rolling/README.md)
- [Scoring Service](./scoring/README.md)

### Running with Datadog Agent

The services support both OpenTelemetry instrumentation and Datadog native tracing:

```bash
# OpenTelemetry
cd game_controller
./run-otel-local.sh

# Datadog native tracing
./run-dd-local.sh
```

## Observability

The application generates:
- **Traces**: Distributed traces across all services
- **Metrics**: HTTP duration, request rate, error rate, custom metrics
- **Logs**: Service logs with trace correlation (when configured)

### Key Metrics

- `http.server.request.duration` - Server-side HTTP request duration
- `http.client.request.duration` - Client-side HTTP request duration
- `games.played` - Custom counter for total games played

## Troubleshooting

### Collector not sending data
1. Verify `DD_API_KEY` is set correctly
2. Check `DD_SITE` matches your Datadog region
3. Review collector logs: `docker-compose logs otelcol`

### Services not connecting
- Ensure all services are healthy: `docker-compose ps`
- Check service logs for connection errors

### No metrics in Datadog
- Verify the collector is running and healthy
- Check that metrics are being generated: `docker-compose logs game_controller`
- Ensure your API key has the correct permissions

## Additional Resources

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Datadog OpenTelemetry Support](https://docs.datadoghq.com/opentelemetry/)
- [Helm Chart Documentation](./helm-chart/README.md)
