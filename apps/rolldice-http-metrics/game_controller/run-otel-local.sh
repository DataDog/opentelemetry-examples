#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export OTEL_METRICS_EXPORTER="otlp"
export OTEL_LOGS_EXPORTER="otlp"
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"
export OTEL_METRIC_EXPORT_INTERVAL=5000
export OTEL_METRIC_EXPORT_TIMEOUT=3000

# Define the path to the Java agent JAR
JAVA_AGENT_JAR="$SCRIPT_DIR/opentelemetry-javaagent.jar"

# Download the Java agent JAR only if it does not exist locally
if [ ! -f "$JAVA_AGENT_JAR" ]; then
	echo "Java agent JAR not found, downloading..."
	curl -L -o "$JAVA_AGENT_JAR" https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
else
	echo "Java agent JAR already exists, skipping download."
fi

export OTEL_RESOURCE_ATTRIBUTES="service.name=game-controller,service.version=1.0,deployment.environment.name=demo-env,host.name=demo-host"

# Build the application first
$SCRIPT_DIR/gradlew build -x test

java -javaagent:$JAVA_AGENT_JAR \
	-jar $SCRIPT_DIR/build/libs/game_controller-0.0.1-SNAPSHOT.jar