# Game Controller

This is an Express.js application that orchestrates the rolldice game. It uses the OpenTelemetry Node.js SDK for auto-instrumentation with W3C TraceContext propagation.

## Prerequisites

* Node.js (v20 or later)
* npm (v9 or later)

## Installation

```bash
npm install
```

## Run the server (standalone)

Set the required environment variables and start the server:

```bash
export OTEL_SERVICE_NAME=game-controller
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
node controller.js
```

The OTel SDK is initialized programmatically in `controller.js` before Express loads, which ensures all HTTP requests are automatically instrumented.
