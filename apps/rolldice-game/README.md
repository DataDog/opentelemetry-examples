# Rolldice Game - W3C Trace Context Example

This project demonstrates **distributed tracing with W3C Trace Context propagation** across three microservices instrumented with OpenTelemetry, exporting to Datadog via the OTel Collector.

## Architecture

```
Client --> game_controller (Node.js/Express) --> rolling (Python/Flask)
                                             --> scoring (Python/Flask)
```

- **game_controller**: Node.js Express service that orchestrates the game. Calls the rolling and scoring services.
- **rolling**: Python Flask service that simulates a dice roll and tracks roll metrics.
- **scoring**: Python Flask service that tracks player scores.

All services export traces, metrics, and logs via OTLP to an OpenTelemetry Collector, which forwards them to Datadog.

## Prerequisites

- Docker and Docker Compose
- A [Datadog API key](https://app.datadoghq.com/organization-settings/api-keys)

## Quick Start (Docker)

1. Copy the environment file and set your Datadog API key:
   ```bash
   cp .env.example .env
   # Edit .env and set DD_API_KEY
   ```

2. Start all services:
   ```bash
   docker compose up --build
   ```

3. Trigger the service call:

   **Success:**
   ```bash
   curl -X POST http://localhost:5002/play_game \
        -H "Content-Type: application/json" \
        -d '{"player": "John Doe"}'
   ```

   **Error (missing player):**
   ```bash
   curl -X POST http://localhost:5002/play_game \
        -H "Content-Type: application/json" \
        -d '{}'
   ```

4. View traces and metrics in [Datadog APM](https://app.datadoghq.com/apm/traces).

## Standalone Host Setup

1. Install and configure the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/installation/) on your host.
2. Use the provided [config.yml](./config.yml) as your collector configuration. Set `DD_API_KEY` in your environment.
3. Start the collector: `./otelcol-contrib --config=config.yml`
4. Start each service (see individual service READMEs for instructions).

## Trace Context Propagation

This example demonstrates W3C TraceContext propagation between services:

1. The **game_controller** receives a request and starts a trace.
2. When calling the **rolling** service, W3C `traceparent` headers are automatically injected by the OTel HTTP instrumentation.
3. The **rolling** service extracts the trace context from incoming headers, creating child spans under the same trace.
4. The same propagation happens when the **game_controller** calls the **scoring** service.

This results in a single distributed trace visible in Datadog APM that spans all three services.

## Configuration

All OTel configuration is done via environment variables (see [docker-compose.yml](./docker-compose.yml)):

| Variable | Description |
|----------|-------------|
| `OTEL_SERVICE_NAME` | Sets the service name in Datadog APM |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTel Collector endpoint |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | Transport protocol (grpc) |
| `OTEL_PROPAGATORS` | Context propagation format (tracecontext,baggage) |
| `DD_API_KEY` | Datadog API key for the collector |
| `DD_SITE` | Datadog site (default: datadoghq.com) |
