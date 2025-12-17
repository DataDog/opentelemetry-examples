# calendar-go

calendar-go is a HTTP service that demonstrates how to capture traces, metrics, and logs in a Go application instrumented with OpenTelemetry.

## Running Locally

### Direct OTLP Ingest to Datadog

Use the provided script to run the application with direct OTLP ingest to Datadog:

```bash
# Set your Datadog API key
export DD_API_KEY="your-api-key"

# Run the application
./run-otel-ingest.sh
```

The script configures the following environment variables:

- `OTEL_EXPORTER_OTLP_PROTOCOL` - Protocol for OTLP export (`http/protobuf` or `grpc`)
- `OTEL_EXPORTER_OTLP_ENDPOINT` - Datadog OTLP endpoint
- `OTEL_EXPORTER_OTLP_HEADERS` - API key and source headers
- `OTEL_RESOURCE_ATTRIBUTES` - Resource attributes including service name

````

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `9090` |
| `OTEL_SERVICE_NAME` | Service name for telemetry | `calendar-rest-go` |
| `OTEL_RESOURCE_ATTRIBUTES` | Resource attributes (e.g., `service.name=foo,service.version=1.0`) | - |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | OTLP protocol (`grpc` or `http/protobuf`) | `http/protobuf` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint | - |
| `OTEL_EXPORTER_OTLP_HEADERS` | Headers for OTLP requests | - |

## Docker Build

This application can be built with the following command:

```bash
docker build -t <TAG_NAME> . --platform linux/amd64
````

## Deploying

After building the Docker image, the tag can be pushed and added into `k8s/deployment.yaml` to be deployed with Kubernetes.
