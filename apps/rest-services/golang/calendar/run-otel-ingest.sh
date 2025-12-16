#! /bin/sh

export OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
export OTEL_EXPORTER_OTLP_ENDPOINT="https://otlp.us5.datadoghq.com"

export OTEL_EXPORTER_OTLP_HEADERS="dd-api-key=${DD_API_KEY},dd-otlp-source=datadog"
export OTEL_RESOURCE_ATTRIBUTES="service.name=my-calendar-service,service.version=1.0,deployment.environment.name=my-otel-test,host.name=my-calendar-host"

go run .
