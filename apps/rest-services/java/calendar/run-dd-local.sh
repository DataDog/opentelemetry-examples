#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export DD_SERVICE="calendar-dd"
export DD_ENV="docker"
export DD_VERSION="1.0-beta"
export DD_TRACE_OTEL_ENABLED="true"
export DD_TRACE_PROPAGATION_STYLE_INJECT="tracecontext"
export DD_TRACE_PROPAGATION_STYLE_EXTRACT="tracecontext"
export SERVER_PORT="9090"
export DD_AGENT_HOST="localhost"
# export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobufç
curl -Lo dd-java-agent.jar https://dtdg.co/latest-java-tracer

java -javaagent:$SCRIPT_DIR/dd-java-agent.jar \
	-jar $SCRIPT_DIR/build/libs/calendar-0.0.1-SNAPSHOT.jar
