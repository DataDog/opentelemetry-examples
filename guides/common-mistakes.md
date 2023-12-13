# Common OpenTelemetry setup mistakes

This document lists the most common mistakes when setting up an OpenTelemetry pipeline to send data to Datadog.

## 1. Using the incorrect OpenTelemetry distro

The Datadog exporter is not present in all OpenTelemetry distros, for example, the *core* OpenTelemetry distro does not include
vendor-specific exporters. It is common for people to confuse in which distro our exporter is present.

For example, people may try to use the `otel/opentelemetry-collector` Docker image instead of the `otel/opentelemetry-collector-contrib` image.
This will make the Collector fail to start.

## 2. Forgetting to include defined components in the pipeline

The OpenTelemetry Collector does not error out if a component is defined but not used in any pipeline.
In the following configuration, the OTLP receiver is configured but not used in any pipeline:

```yaml
receivers:
    otlp:
      protocols:
        grpc:
    statsd:

exporters:
  datadog:
    api:
      key: ${DD_API_KEY}

service:
  pipelines:
    metrics:
      receivers: [statsd] # MISSING otlp
      exporters: [datadog]
```

If the component is not added to the pipeline, it won't work.
