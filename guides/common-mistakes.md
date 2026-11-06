# Common OpenTelemetry setup mistakes

This document lists common mistakes when setting up an OpenTelemetry pipeline that sends data to Datadog.

## 1. Using the incorrect OpenTelemetry distro

Not every OpenTelemetry Collector [distribution](https://opentelemetry.io/docs/collector/distributions/) includes the Datadog exporter.
The *core* distribution excludes vendor-specific exporters.

For example, if you configure the Datadog exporter with the `otel/opentelemetry-collector` Docker image, the Collector fails to start.
Use the `otel/opentelemetry-collector-contrib` image instead.

## 2. Forgetting to include defined components in the pipeline

The OpenTelemetry Collector only enables components that are part of a pipeline, and it doesn't report an error for a component that no pipeline uses.
See the [`service` section](https://opentelemetry.io/docs/collector/configuration/#service) of the Collector configuration docs.

In the following configuration, the OTLP receiver is defined but missing from the `metrics` pipeline:

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
