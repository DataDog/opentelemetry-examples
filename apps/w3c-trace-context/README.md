## W3C Trace Context Propagation Example

Demonstrates [W3C Trace Context](https://www.w3.org/TR/trace-context/) propagation
between OpenTelemetry-instrumented and Datadog-instrumented services. A Java
(Spring Boot) "frontend" calendar service calls a Python (Flask) "backend"
calendar service. Both agents -- OTel and Datadog -- inject and extract
`traceparent` and `tracestate` HTTP headers automatically so that a single
distributed trace spans both services regardless of which instrumentation
library each one uses.

### API

```
GET /calendar   -> returns a random date in 2022
GET /health     -> health check endpoint
```

Request:
```bash
curl http://localhost:8080/calendar
```

Response:
```json
{"date":"3/22/2022"}
```

### Prerequisites

Set your Datadog API key:

```bash
export DD_API_KEY=<your-api-key>
```

Optionally copy `.env.example` to `.env` and fill in the values.

### Deployment Modes

The example ships three docker-compose files, each showing a different
combination of OTel and Datadog instrumentation. All three use the Datadog
Agent as the backend for both OTLP and DD trace intake.

#### Mode 1 -- OTel Java, DD Python

Java service instrumented with the OTel Java agent sends traces via OTLP.
Python service instrumented with ddtrace sends traces via the DD trace API.
W3C `traceparent`/`tracestate` headers link the two.

```bash
docker compose -f docker-compose-otel-java-dd-py.yaml up
```

| Service | Port | Instrumentation |
|---------|------|-----------------|
| calendar-java-otel | 8080 | OpenTelemetry Java agent |
| calendar-py-dd | 9090 | ddtrace |

#### Mode 2 -- DD Java, OTel Python

Java service instrumented with dd-java-agent; Python service instrumented
with the OTel Python SDK. `DD_TRACE_PROPAGATION_STYLE=tracecontext` on the
Java side ensures W3C headers are emitted.

```bash
docker compose -f docker-compose-dd-java-otel-py.yaml up
```

| Service | Port | Instrumentation |
|---------|------|-----------------|
| calendar-java-dd | 8080 | dd-java-agent |
| calendar-py-otel | 9090 | OpenTelemetry Python |

#### Mode 3 -- Both side by side

Two parallel request chains through a single Datadog Agent, proving
cross-instrumentation interoperability in both directions.

```bash
docker compose -f docker-compose-dd-otel-both.yaml up
```

| Service | Port | Instrumentation |
|---------|------|-----------------|
| calendar-java-dd (chain A) | 8080 | dd-java-agent |
| calendar-py-otel (chain A) | 9091 | OpenTelemetry Python |
| calendar-java-otel (chain B) | 8081 | OpenTelemetry Java agent |
| calendar-py-dd (chain B) | 9090 | ddtrace |

### How W3C Trace Context works here

Both the OpenTelemetry agents and the Datadog agents handle `traceparent`
and `tracestate` injection/extraction automatically:

- **OTel services** use `OTEL_PROPAGATORS=tracecontext,baggage` (the default)
  to propagate W3C headers.
- **DD services** use `DD_TRACE_PROPAGATION_STYLE=tracecontext` to switch
  from the Datadog-native propagation format to W3C Trace Context.
- **128-bit trace IDs** are enabled on DD services via
  `DD_TRACE_128_BIT_TRACEID_GENERATION_ENABLED=true` so that trace IDs are
  compatible with the 32-hex-character trace-id field in the `traceparent`
  header.

The Datadog Agent accepts both OTLP (ports 4317/4318) and DD APM traces
(port 8126), correlating them into unified traces in the Datadog UI.

### Configuration reference

| Variable | Used by | Purpose |
|----------|---------|---------|
| `DD_API_KEY` | DD Agent | Datadog API key |
| `DD_SITE` | DD Agent | Datadog site (default `datadoghq.com`) |
| `DD_TRACE_PROPAGATION_STYLE` | DD services | Set to `tracecontext` for W3C |
| `DD_TRACE_128_BIT_TRACEID_GENERATION_ENABLED` | DD services | Full 128-bit trace IDs |
| `OTEL_PROPAGATORS` | OTel services | Propagator chain (`tracecontext,baggage`) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTel services | OTLP endpoint on the DD Agent |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | OTel services | `grpc` or `http/protobuf` |
