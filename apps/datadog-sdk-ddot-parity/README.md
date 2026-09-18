# Datadog SDK trace-export parity validation

This example compares the same Datadog SDK applications and workloads with
two trace-export paths:

- **DDOT setup:** the SDK exports OTLP traces to standalone DDOT. A Datadog
  Agent runs beside it for profiling, Dynamic Instrumentation, DogStatsD, and
  PostgreSQL Database Monitoring.
- **Datadog native export:** the same SDK sends native Msgpack traces directly
  to the Agent on port 8126. The application build, service identity, Agent,
  database, and workload remain unchanged.

The applications cover .NET, Go, Java, Node.js, and Python. Their deterministic
endpoints generate OTel API spans, profiles, debugger data, and database
queries so product behavior can be compared with as few changing variables as
possible.

## Layout

| Path | Purpose |
| --- | --- |
| `*-app/` | Pinned SDK app and container build |
| `k8s/` | One isolated namespace per SDK, standalone DDOT, Agent, and PostgreSQL |
| `scripts/run-*-product-workload.sh` | Repeatable product workload with UTC evidence boundaries |
| `scripts/run-*-scenario.sh` | Smaller scenario windows for focused checks |
| `scripts/set-trace-export.sh` | Switch only the trace transport between DDOT and native export |

## Run a lane

Install the upstream OpenTelemetry Operator and configure it to use a DDOT
collector image. Build the selected app image with the tag used by its
manifest. Then apply its application manifest and, except for .NET where the
collector is embedded in the application manifest, its collector manifest.

Each namespace needs a `datadog-secret` with `api-key` and `site` keys. For
example:

```shell
kubectl create namespace otel-python-validation --dry-run=client -o yaml | kubectl apply -f -
kubectl -n otel-python-validation create secret generic datadog-secret \
  --from-literal=api-key="$DD_API_KEY" \
  --from-literal=site="$DD_SITE"
kubectl apply -f k8s/python-operator-validation.yaml
kubectl apply -f k8s/ddot-collector.yaml
```

Wait for the app, standalone DDOT, Agent, and PostgreSQL pods to be ready.
Forward the application service, then run its workload script. The Kubernetes
guide lists the namespace, manifest, local port, and script for every SDK.

## Compare the export paths

Run the DDOT setup first, record the UTC window printed by the workload, and
confirm that the service is receiving traces. Then switch only the transport:

```shell
scripts/set-trace-export.sh python native
```

After the rollout, run the identical workload again. This sends Datadog native
Msgpack traces to the Agent. Restore standalone DDOT export with:

```shell
scripts/set-trace-export.sh python ddot
```

The Python app supplies an OTLP default in code, so its native switch also sets
`DD_TRACE_AGENT_PROTOCOL_VERSION=v0.4`. Restoring DDOT removes that override.

Use unique time windows and keep the service, environment, version, test data,
and workload volume equal across the two runs.

## Interpreting parity

Sampling differs across products, so trace correlation may not appear on every
profile, debugger record, or database sample. Generate enough deterministic
traffic to find representative correlated examples; a single missing link is
not sufficient evidence of unsupported behavior.

Record a feature as unsupported only when it works with Datadog native export
and fails specifically through DDOT or OTel ingestion. If the same behavior is
missing in both setups, treat the comparison as not applicable and revisit the
feature definition or expected evidence.
