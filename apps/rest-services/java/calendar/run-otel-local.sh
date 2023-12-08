#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export OTEL_METRICS_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
# export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
curl -LO https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
export OTEL_RESOURCE_ATTRIBUTES=service.name=my-calendar-service,deployment.environment=otel-test,host.name=calendar-host

java -javaagent:$SCRIPT_DIR/opentelemetry-javaagent.jar \
	-jar $SCRIPT_DIR/build/libs/calendar-0.0.1-SNAPSHOT.jar
