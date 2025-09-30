#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export OTEL_EXPORTER_OTLP_TRACES_PROTOCOL="http/protobuf"
export OTEL_EXPORTER_OTLP_LOGS_PROTOCOL="http/protobuf"
export OTEL_EXPORTER_OTLP_ENDPOINT="https://otlp.us5.datadoghq.com/"

export OTEL_EXPORTER_OTLP_HEADERS="dd-api-key=${DD_API_KEY},dd-otlp-source=datadog"
# export OTEL_EXPORTER_OTLP_TRACES_HEADERS="DD-CLIENT-TOKEN=${DD_CLIENT_TOKEN}, dd-otlp-source=datadog"

# Define the path to the Java agent JAR
JAVA_AGENT_JAR="$SCRIPT_DIR/opentelemetry-javaagent.jar"

# Download the Java agent JAR only if it does not exist locally
if [ ! -f "$JAVA_AGENT_JAR" ]; then
    echo "Java agent JAR not found, downloading..."
    curl -L -o "$JAVA_AGENT_JAR" https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
else
    echo "Java agent JAR already exists, skipping download."
fi

export OTEL_RESOURCE_ATTRIBUTES="service.name=my-calendar-service,service.version=1.0,deployment.environment.name=my-otel-test,host.name=my-calendar-host"

java -javaagent:$JAVA_AGENT_JAR \
    -jar $SCRIPT_DIR/build/libs/calendar-0.0.1-SNAPSHOT.jar
