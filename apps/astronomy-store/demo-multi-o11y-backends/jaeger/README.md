# jaeger

Deploys a single-node, in-memory Jaeger instance for local development, using the official
[`jaegertracing/jaeger`][chart] Helm chart. All-in-one binary (Jaeger v2), no external storage
dependency — traces are held in memory and lost whenever the pod restarts.

## What this deploys

The chart's default config assumes Elasticsearch. `values.yaml` overrides it (via `userconfig`)
with a self-contained Jaeger v2 pipeline:

- **`otlp` receiver** — accepts traces over OTLP gRPC (4317) and HTTP (4318).
- **`jaeger_storage` extension** — an in-memory `memstore` backend, capped at 100,000 traces.
- **`jaeger_query` extension** — serves the UI (16686) and gRPC query API (16685) from `memstore`.
- **`healthcheckv2` extension** — backs the chart's default liveness/readiness probes.

## Prerequisites

- `helm` and `kubectl`, pointed at the target cluster.

## Install

```sh
./install
```

This creates the `jaeger` namespace (if needed), adds the `jaegertracing` Helm repo, and
installs/upgrades the `jaeger` release from `values.yaml`.

## Send traces to it

From inside the cluster:

```
grpc://jaeger.jaeger.svc.cluster.local:4317
http://jaeger.jaeger.svc.cluster.local:4318
```

## View traces

```sh
kubectl port-forward -n jaeger svc/jaeger 16686:16686
open http://localhost:16686
```

## Uninstall

```sh
./uninstall
```

Removes the `jaeger` Helm release (and all in-memory trace data with it). The `jaeger`
namespace itself is left in place.

[chart]: https://github.com/jaegertracing/helm-charts/tree/main/charts/jaeger
