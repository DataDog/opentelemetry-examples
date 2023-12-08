# calendar-go
calendar-go is a HTTP service that demonstrates how to capture various metrics in a Go application instrumented with OTel.

## Docker Build

This application can be built with the following command:

```
docker build -t <TAG_NAME> . --platform linux/amd64
```

## Deploying

After building the Docker image, the tag can be pushed and added into `k8s/deployment.yaml` to be deployed with Kubernetes.
