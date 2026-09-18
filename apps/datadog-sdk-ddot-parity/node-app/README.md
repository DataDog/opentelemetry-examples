# Node.js OTLP product validation app

This app pins Node.js 22 and `dd-trace` 6.16.0. Traces use the Datadog tracer's
OTLP HTTP/JSON exporter and a standalone DDOT collector. Profiling, Dynamic
Instrumentation, and DBM propagation use their normal Agent side channels.

The workload endpoints produce deterministic OTel-API spans, debugger captures,
CPU work, retained allocations, caught exceptions, and PostgreSQL traffic.
