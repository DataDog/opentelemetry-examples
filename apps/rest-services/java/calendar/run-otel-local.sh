#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export OTEL_METRICS_EXPORTER="otlp"
export OTEL_LOGS_EXPORTER="otlp"
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"

# Define the path to the Java agent JAR
JAVA_AGENT_JAR="$SCRIPT_DIR/opentelemetry-javaagent.jar"

# Download the Java agent JAR only if it does not exist locally
if [ ! -f "$JAVA_AGENT_JAR" ]; then
	echo "Java agent JAR not found, downloading..."
	curl -L -o "$JAVA_AGENT_JAR" https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
else
	echo "Java agent JAR already exists, skipping download."
fi

export OTEL_RESOURCE_ATTRIBUTES="service.name=my-calendar-service,service.version=1.0,deployment.environment.name=otel-test,host.name=calendar-host"

java -javaagent:$JAVA_AGENT_JAR \
	-jar $SCRIPT_DIR/build/libs/calendar-0.0.1-SNAPSHOT.jar
