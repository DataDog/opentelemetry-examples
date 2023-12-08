# Intro to OpenTelemetry

[OpenTelemetry][1] is a collection of standards, libraries, Datadog Agent-like daemons and conventions that define a vendor-neutral way of gathering, sending and processing telemetry signals (metrics, traces, logs, ...).
It supersedes the [OpenMetrics][2] and [OpenTracing][3] projects, and it is used by cloud providers (Google Cloud, AWS...) and competitors (Splunk) as the default way of collecting and sending telemetry data.

Some important components of OpenTelemetry are the following:
- The OTLP protocol (**O**pen**T**e**L**emetry **P**rotocol) for sending telemetry data (currently metrics, traces and logs). It defines a common data structure that can be sent through gRPC/Protobuf, HTTP/Binary or HTTP/JSON.
- The OpenTelemetry language instrumentation libraries, that define an API and SDK for sending telemetry data in the OTLP and other formats.
- The OpenTelemetry Collector a vendor-neutral Agent for consuming telemetry data through various protocols (most notably OTLP) and exporting it to different backends.
  It comes in different [distributions][4] e.g. community supported (OpenTelemetry Collector contrib) or vendor-supported (AWS OpenTelemetry Collector distro or Splunk OpenTelemetry Collector distro).

Datadog provides support for OpenTelemetry by having

- a [Datadog exporter][5] in the OpenTelemetry Collector contrib and AWS OpenTelemetry distributions, that can currently support metrics and traces, and
- support for OTLP metrics and traces ingestion through the Datadog Agent (currently experimental).


## OpenTelemetry Collector basics

The [OpenTelemetry Collector][6] works by defining data *pipelines* that 
- ingest data in a given format via *receivers* and transform it to the OTLP format,
- apply transformations to this data using *processors* and
- transform the data to a given backend format and send it to the backend via *exporters*.

It is written in Go, is cross-platform, and its configuration is written in YAML. 

An example minimal configuration for receiving metrics in OTLP format via gRPC and exporting them to Datadog is:

```yaml
receivers:
  otlp:
    protocols:
      grpc:

processors:
  batch:
   timeout: 10s

exporters:
  datadog:
    api:
      key: ${DD_API_KEY}

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [datadog]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [datadog]
```

More detailed examples are available in the `collector-configurations` folder.

There are different OpenTelemetry Collector *distros* that support different receivers, processors and exporters, as well as other cross-cutting features (e.g. configuration sources).
A distro is able to export data to Datadog if the `datadog` exporter is available. We are currently present in two distros:

- the [OpenTelemetry Collector contrib distro][7], which is OpenTelemetry community supported, and
- the [AWS Distro for OpenTelemetry (ADOT)][8] which is AWS supported.

When vendors do not offer native OTLP ingest (that is, they don't expose a public OTLP endpoint in their backend), the Collector is the OpenTelemetry-supported way of getting data from instrumentation libraries into the vendor backend.

## Datadog Agent OTLP support basics

The Datadog Agent has since version 7.31 experimental support for OTLP traces, and, starting on version 7.32 it will have experimental support for OTLP metrics too.
It is more limited than the OpenTelemetry Collector today but simpler to configure. To enable it a minimal configuration (equivalent to the example above) would be:

```yaml
api_key: <API key>

experimental:
  otlp:
    grpc_port: 4317
```


[1]: https://opentelemetry.io
[2]: https://openmetrics.io
[3]: https://opentracing.io
[4]: https://opentelemetry.io/docs/concepts/distributions
[5]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/datadogexporter#datadog-exporter
[6]: https://opentelemetry.io/docs/collector
[7]: https://github.com/open-telemetry/opentelemetry-collector-contrib
[8]: https://aws-otel.github.io
