# Manual Container Metrics Application
This project consists of a Go server instrumented with OpenTelemetry with the OpenTelemetry Collector.

The server manually creates metrics with the same names as container metrics from the docker runtime. These metrics are assigned the same `container.id` and `container.name` as the server to demonstrate that trace container metrics correlation works in the trace app. Traces will be automatically generated from the Kubernetes liveness and readiness requests.

## Docker Build
This application can be built with the following command:
```
docker build -t <TAG_NAME> . --platform linux/amd64
```

## Deploying
After building the Docker image, the tag can be pushed and added into `values.yaml` to be deployed with Kubernetes.
