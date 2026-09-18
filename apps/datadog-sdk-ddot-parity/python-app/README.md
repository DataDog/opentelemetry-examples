# Python OTLP product validation app

This Flask app pins `dd-trace-py` 4.15.0. In the DDOT setup, traces use OTLP
HTTP/protobuf while profiling, Dynamic Instrumentation, and DBM propagation use
their normal Agent side channels.

## Endpoints (port 5000 inside the container)

| Route     | Signal                                              |
|-----------|-----------------------------------------------------|
| `/span`   | OTel `Tracer` -> `manual.span`                      |
| `/metric` | OTel `Meter` counter `demo.requests_total`          |
| `/log`    | stdlib `logging.info(...)` bridged via OTel logging |
| `/probe` | Stable function for Dynamic Instrumentation probes |
| `/di/log` | Log probe without snapshot |
| `/di/snapshot-line` | Line snapshot probe |
| `/di/snapshot-method` | Method snapshot probe |
| `/di/metrics` | Count, gauge, histogram, and distribution probes |
| `/di/span` | Dynamic span probe |
| `/di/decorate` | Active-span decoration and code-origin workload |
| `/cpu` | CPU workload for profiling |
| `/deep-recursion` | Deterministic deep stack for profile integrity |
| `/thread-contention` | Named worker threads and lock contention |
| `/async-wait` | Deterministic async wall-time workload |
| `/allocate` | Retained allocations for profiling |
| `/exceptions` | Caught exceptions for profiling |
| `/db/query` | Parameterized PostgreSQL query for DBM correlation |
| `/db/slow` | Slow PostgreSQL query for samples and plans |
| `/db/error` | Intentional PostgreSQL error |

`probes.json` installs line and method snapshots, four metric kinds, a
conditional probe, a dynamic span, active-span decoration, and a log probe
without requiring UI setup.
This keeps the submission tests reproducible; Remote Configuration remains
enabled so the same service can also be tested with probes created in Datadog.

See [`../README.md`](../README.md) for deployment and trace-export switching
instructions.
