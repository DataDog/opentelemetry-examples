# Datadog Agent Configurations

This folder contains a minimal Datadog Agent configuration, `values.yaml`, with [OTLP Ingestion](https://docs.datadoghq.com/opentelemetry/otlp_ingest_in_the_agent/?tab=host) enabled. This file enables the `DD_OTLP_CONFIG_TRACES_SPAN_NAME_AS_RESOURCE_NAME` environment variable in order to use the OpenTelemetry span name as the Datadog resource name.

For more detail regarding Datadog Agent OTLP Ingestion settings, see the [config template](https://github.com/DataDog/datadog-agent/blob/7.49.0/pkg/config/config_template.yaml#L3804-L4062).
