# Manual Container Metrics Application

This project consists of a Go server instrumented with OpenTelemetry, exporting via the OTLP protocol to the OTel Collector (or Datadog Agent with OTLP ingestion enabled).

The server manually creates metrics with the same names as container metrics from the Docker/containerd runtime. These metrics are assigned the same `container.id` and `container.name` as the server container to demonstrate that trace-to-container-metrics correlation works in the Datadog APM trace view. Traces are automatically generated from Kubernetes liveness and readiness probe requests.

## Container Metrics

The following container metrics are emitted as OTel Gauge instruments:

| Metric Name | Unit | Description |
|---|---|---|
| `container.cpu.usage` | ns | Total CPU usage in nanoseconds |
| `container.cpu.limit` | {cpus} | CPU limit assigned to the container |
| `container.cpu.user` | ns | User CPU time in nanoseconds |
| `container.cpu.system` | ns | System CPU time in nanoseconds |
| `container.memory.rss` | By | Resident set size memory in bytes |
| `container.memory.usage` | By | Total memory usage in bytes |
| `container.memory.limit` | By | Memory limit in bytes |
| `container.io.read` | By | Bytes read from disk |
| `container.io.write` | By | Bytes written to disk |
| `container.net.sent` | By | Bytes sent over network |
| `container.net.rcvd` | By | Bytes received over network |

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `OTEL_SERVICE_NAME` | Service name for OTel resource | Yes |
| `OTEL_CONTAINER_NAME` | Container name for metric correlation | Yes |
| `OTEL_K8S_CONTAINER_ID` | Container/pod ID for metric correlation | Yes |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint (e.g., `http://host:4317`) | Yes |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | OTLP protocol (`grpc`) | Yes |
| `OTEL_RESOURCE_ATTRIBUTES` | Additional OTel resource attributes | No |

## Endpoints

| Path | Description |
|---|---|
| `/health` | Health check (returns `{"status":"healthy"}`) |
| `/readiness` | Readiness probe with trace correlation |
| `/liveness` | Liveness probe with trace correlation |

## Docker Build

Build the application image:

```bash
docker build -t <TAG_NAME> . --platform linux/amd64
```

## Deploying to Kubernetes

After building and pushing the Docker image, update the image tag in `values.yaml` and apply:

```bash
kubectl apply -f values.yaml
```

The deployment expects an OTel Collector or Datadog Agent running on each node with OTLP ingestion enabled on port 4317.
