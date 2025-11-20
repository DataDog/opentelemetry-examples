#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export DD_SERVICE="game-controller-dd"
export DD_ENV="docker"
export DD_VERSION="1.0-beta"
export DD_TRACE_OTEL_ENABLED="true"
export DD_TRACE_PROPAGATION_STYLE_INJECT="tracecontext"
export DD_TRACE_PROPAGATION_STYLE_EXTRACT="tracecontext"
export PORT="5002"
export DD_AGENT_HOST="localhost"

# Define the path to the Datadog Java agent JAR
DD_AGENT_JAR="$SCRIPT_DIR/dd-java-agent.jar"

# Download the Datadog agent JAR only if it does not exist locally
if [ ! -f "$DD_AGENT_JAR" ]; then
	echo "Datadog agent JAR not found, downloading..."
	curl -Lo "$DD_AGENT_JAR" https://dtdg.co/latest-java-tracer
else
	echo "Datadog agent JAR already exists, skipping download."
fi

# Build the application first
$SCRIPT_DIR/gradlew build -x test

java -javaagent:$DD_AGENT_JAR \
	-jar $SCRIPT_DIR/build/libs/game_controller-0.0.1-SNAPSHOT.jar