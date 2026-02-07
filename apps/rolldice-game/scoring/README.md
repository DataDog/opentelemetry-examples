# Score Server

This is a Flask application that tracks player scores. It uses the OpenTelemetry Python SDK for tracing, with auto-instrumentation via the `opentelemetry-instrument` CLI.

## Prerequisites

* Python (v3.12+)
* pip

## Installation

```bash
pip install -r requirements.txt
```

## Run the server (standalone)

```bash
export OTEL_SERVICE_NAME=scoring
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
opentelemetry-instrument --service_name scoring --logs_exporter otlp flask run -p 5001
```

The `opentelemetry-instrument` CLI auto-configures the OTel SDK from environment variables and instruments Flask to capture HTTP spans and propagate W3C TraceContext.
