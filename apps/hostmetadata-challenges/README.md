# Host Metadata Challenges

Demonstrates how a standard Kubernetes OpenTelemetry deployment produces **three independent OTLP payload shapes** — each carrying a different slice of host and workload identity — because the data originates from separate receivers and pipelines that never share a `Resource`.

## The Problem

A single Kubernetes workload has identity spread across several layers:

| Layer | Example attributes | Where it comes from |
|-------|-------------------|---------------------|
| **Workload** | `k8s.namespace.name`, `k8s.pod.name`, `k8s.pod.uid`, `k8s.deployment.name`, `k8s.node.name` | [k8sattributes processor][2] enriching app-emitted telemetry |
| **Host** | `host.name`, `host.id`, `host.ip`, `host.mac` | [hostmetrics receiver][4] scraping `/proc` and `/sys` on the node |
| **Object state** | `namespace`, `pod`, `node`, `created_by_kind`, `host_ip`, `pod_ip` | [kube-state-metrics][5] querying the Kubernetes API |

In an OpenTelemetry Collector setup that follows the [recommended Kubernetes architecture][1] — a **DaemonSet** for node-local collection plus a **Deployment** for cluster-wide collection — these three data sources feed into separate pipelines. Each pipeline produces its own `ResourceSpans` or `ResourceMetrics` with its own `Resource` block. Nothing merges them automatically.

This means a backend receiving this telemetry sees three different resources for the same workload, each with different attributes. This demo makes that separation concrete and inspectable.

## Architecture

There are five Kubernetes objects in the `otel-multi-pipeline-demo` namespace:

```
┌──────────────────────────────────────────────────────────────────┐
│ Node                                                             │
│                                                                  │
│  ┌─────────────────────┐        ┌──────────────────────────────┐ │
│  │ trace-generator pod │ OTLP   │ OTel Agent pod (DaemonSet)   │ │
│  │                     │───────>│                              │ │
│  │ Go app emitting     │ gRPC   │ ┌──────────┐ ┌────────────┐ │ │
│  │ spans via OTel SDK  │ :4317  │ │ Pipeline │ │ Pipeline   │ │ │
│  └─────────────────────┘        │ │ traces   │ │ metrics    │ │ │
│                                 │ │          │ │            │ │ │
│       /proc, /sys ─────────────>│ │ otlp rx  │ │ hostmetrics│ │ │
│       (host filesystem)         │ │    ↓     │ │ rx         │ │ │
│                                 │ │ k8sattr  │ │    ↓       │ │ │
│                                 │ │    ↓     │ │  batch     │ │ │
│                                 │ │  batch   │ │    ↓       │ │ │
│                                 │ │    ↓     │ │  otlp tx   │ │ │
│                                 │ │ otlp tx  │ │ ──────┐    │ │ │
│                                 │ │ ───┐     │ │       │    │ │ │
│                                 │ └────│─────┘ └───────│────┘ │ │
│                                 └──────│───────────────│──────┘ │
└────────────────────────────────────────│───────────────│────────┘
                                         │               │
                              OTLP/gRPC  │               │  OTLP/gRPC
                            (ResourceSpans)        (ResourceMetrics)
                                         │               │
                                         ▼               ▼
┌──────────────────────────────────────────────────────────────────┐
│ OTel Gateway pod (Deployment)                                    │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │ otlp receiver (:4317)                                    │    │
│  │   receives forwarded traces (with k8s.* attrs)           │    │
│  │   receives forwarded metrics (with host.* attrs)         │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │ prometheus receiver                                      │    │
│  │   scrapes kube-state-metrics :8080 every 15s             │    │
│  │   produces ResourceMetrics with k8s object-state labels  │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                  │
│  All three sources ──> batch processor ──> debug exporter        │
│  (printed to stdout with full Resource attributes)               │
└──────────────────────────────────────────────────────────────────┘
        ▲
        │ Prometheus scrape :8080/metrics
        │
┌───────┴──────────────────────┐
│ kube-state-metrics pod       │
│ (Deployment)                 │
│                              │
│ Queries K8s API for object   │
│ state: pods, nodes,          │
│ deployments, replicasets     │
│                              │
│ Exposes as Prometheus        │
│ metrics with labels like:    │
│   namespace, pod, node,      │
│   created_by_kind, host_ip   │
└──────────────────────────────┘
```

### How the three payload shapes are created

**Payload 1 — App traces with workload identity:**

```
trace-generator pod
    │
    │  Emits OTLP spans over gRPC to the agent on the same node.
    │  The spans carry only what the app SDK sets: service.name,
    │  service.version, telemetry.sdk.*, and the span data itself.
    │
    ▼
OTel Agent ── traces pipeline
    │
    │  k8sattributes processor sees the source connection IP,
    │  resolves it to a pod via the Kubernetes API, and adds:
    │    k8s.namespace.name
    │    k8s.pod.name
    │    k8s.pod.uid
    │    k8s.deployment.name
    │    k8s.node.name
    │
    │  These go into the Resource block of the ResourceSpans.
    │  No host.* attributes are added — the processor only
    │  resolves Kubernetes workload identity.
    │
    ▼
OTel Gateway ── receives ResourceSpans with k8s.* attributes
    │
    ▼
debug exporter ── prints to stdout
```

**Payload 2 — Host metrics with host identity:**

```
Node filesystem (/proc, /sys)
    │
    │  Mounted into the agent pod via hostPath volumes.
    │
    ▼
OTel Agent ── metrics pipeline
    │
    │  hostmetrics receiver scrapes CPU, memory, filesystem,
    │  network, and load data from the host.
    │
    │  The receiver creates ResourceMetrics whose Resource
    │  contains host.name (always set), and in cloud environments
    │  also host.id, host.ip, host.mac (from the resourcedetection
    │  processor, not included in this minimal setup).
    │
    │  No k8s.* attributes are present — this pipeline has no
    │  k8sattributes processor because there is no originating
    │  pod to resolve (the data comes from the host itself).
    │
    ▼
OTel Gateway ── receives ResourceMetrics with host.* attributes
    │
    ▼
debug exporter ── prints to stdout
```

**Payload 3 — kube-state-metrics with Kubernetes object state:**

```
Kubernetes API server
    │
    │  kube-state-metrics watches the API for object state:
    │  pods, nodes, deployments, replicasets, etc.
    │
    ▼
kube-state-metrics pod ── exposes /metrics endpoint
    │
    │  Metrics like kube_pod_info, kube_node_info,
    │  kube_deployment_status_replicas carry object state
    │  as Prometheus label dimensions:
    │    namespace, pod, node, created_by_kind,
    │    created_by_name, host_ip, pod_ip
    │
    ▼
OTel Gateway ── prometheus receiver
    │
    │  Scrapes :8080/metrics every 15 seconds.
    │  Converts Prometheus metrics to OTLP ResourceMetrics.
    │  The Resource contains job="kube-state-metrics" and
    │  instance="<pod-ip>:8080". The Kubernetes labels appear
    │  as metric data point attributes.
    │
    │  This is a completely independent data path from both
    │  the app traces and the host metrics.
    │
    ▼
debug exporter ── prints to stdout
```

### Why none of these merge

Each `Resource` block in OTLP is scoped to the receiver+pipeline that produced it. The agent's `traces` pipeline creates one `Resource` (with `k8s.*` from k8sattributes). The agent's `metrics` pipeline creates a different `Resource` (with `host.*` from hostmetrics). The gateway's `prometheus` receiver creates a third `Resource` (with `job` and `instance` from the scrape target). No collector component in this setup combines them.

A downstream backend that wants to correlate these three resources for the same workload must do so itself — by matching on shared dimensions like `k8s.node.name` / `host.name`, or `k8s.pod.name` / Prometheus `pod` label.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

## Quick Start

```bash
# Full setup: create kind cluster, build app, deploy everything
make all
```

Or step by step:

```bash
# 1. Create a local kind cluster
make cluster

# 2. Build the trace-generator app and load into kind
make load

# 3. Deploy all manifests
make deploy
```

When deploy finishes, it waits for all pods to be ready and prints:

```
All pods ready. Run 'make logs-gateway' to inspect resource attributes.
```

## Verifying the Three Payload Shapes

### 1. Confirm all pods are running

```bash
kubectl -n otel-multi-pipeline-demo get pods
```

Expected output:

```
NAME                                  READY   STATUS    RESTARTS   AGE
kube-state-metrics-xxxxxxxxxx-xxxxx   1/1     Running   0          90s
otel-agent-xxxxx                      1/1     Running   0          90s
otel-gateway-xxxxxxxxxx-xxxxx         1/1     Running   0          90s
trace-generator-xxxxxxxxxx-xxxxx      1/1     Running   0          90s
```

### 2. Open the gateway logs

The gateway's `debug` exporter prints every piece of telemetry it receives to stdout, including full `Resource attributes` blocks. This is where all three shapes are visible.

```bash
make logs-gateway
```

### 3. Generate additional traces (optional)

The app emits a `background-tick` span every 5 seconds automatically. To generate on-demand traces:

```bash
# Terminal 1
make port-forward

# Terminal 2
make generate-traces
# or:
curl http://localhost:8080/generate
```

### 4. Identify the three resource shapes in the logs

#### Shape 1: App traces — workload metadata

Search the gateway logs for `ResourceSpans` or `ScopeSpans`. The `Resource attributes` section will look like this:

```
Resource SchemaURL: https://opentelemetry.io/schemas/1.26.0
Resource attributes:
     -> service.name: Str(trace-generator)
     -> service.version: Str(0.1.0)
     -> telemetry.sdk.language: Str(go)
     -> telemetry.sdk.name: Str(opentelemetry)
     -> telemetry.sdk.version: Str(1.34.0)
     -> k8s.namespace.name: Str(otel-multi-pipeline-demo)
     -> k8s.pod.name: Str(trace-generator-xxxxxxxxxx-xxxxx)
     -> k8s.pod.uid: Str(xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
     -> k8s.deployment.name: Str(trace-generator)
     -> k8s.node.name: Str(otel-metadata-demo-control-plane)
```

What is present: `k8s.namespace.name`, `k8s.pod.name`, `k8s.pod.uid`, `k8s.deployment.name`, `k8s.node.name` — all injected by the k8sattributes processor.

What is absent: `host.ip`, `host.mac`, `host.id` — these are not added by k8sattributes and the app SDK does not set them.

#### Shape 2: Host metrics — host metadata

Search for `ResourceMetrics` entries that contain metrics like `system.cpu.time`, `system.memory.usage`, `system.filesystem.usage`, or `system.network.io`. Their `Resource attributes` section will look like this:

```
Resource SchemaURL:
Resource attributes:
     -> host.name: Str(otel-metadata-demo-control-plane)
```

What is present: `host.name` (always set by the hostmetrics receiver).

What is absent: `k8s.pod.name`, `k8s.namespace.name`, `k8s.deployment.name` — the hostmetrics receiver scrapes the host filesystem, not pod telemetry, so there is no pod to resolve.

> **Note on `host.ip`, `host.id`, `host.mac`:** In a kind cluster, the hostmetrics receiver only populates `host.name`. In a real cloud environment (AWS, GCP, Azure), adding a [`resourcedetection` processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor) with the appropriate detector (`ec2`, `gcp`, `azure`) to the agent's metrics pipeline would also set `host.id`, `host.ip`, and `host.mac` on the same `ResourceMetrics`. The payload shape is still separate from the app traces — adding `resourcedetection` enriches the host resource, it does not merge it with the workload resource.

#### Shape 3: kube-state-metrics — Kubernetes object state

Search for `ResourceMetrics` entries with `job = "kube-state-metrics"`. Their `Resource attributes` section will look like this:

```
Resource SchemaURL:
Resource attributes:
     -> service.name: Str(otel-multi-pipeline-demo/kube-state-metrics)
     -> service.instance.id: Str(kube-state-metrics.otel-multi-pipeline-demo.svc.cluster.local:8080)
     -> net.host.name: Str(kube-state-metrics.otel-multi-pipeline-demo.svc.cluster.local)
     -> net.host.port: Str(8080)
     -> http.scheme: Str(http)
     -> server.address: Str(kube-state-metrics.otel-multi-pipeline-demo.svc.cluster.local)
     -> server.port: Str(8080)
```

The Kubernetes object state appears as **metric data point attributes** (not resource attributes), because kube-state-metrics exposes them as Prometheus label dimensions on each metric:

```
Metric #42
Descriptor:
     -> Name: kube_pod_info
     -> Unit:
     -> DataType: Gauge
NumberDataPoints #0
     -> namespace: Str(otel-multi-pipeline-demo)
     -> pod: Str(trace-generator-xxxxxxxxxx-xxxxx)
     -> node: Str(otel-metadata-demo-control-plane)
     -> created_by_kind: Str(ReplicaSet)
     -> created_by_name: Str(trace-generator-xxxxxxxxxx)
     -> host_ip: Str(172.18.0.2)
     -> pod_ip: Str(10.244.0.x)
     -> uid: Str(xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)

Metric #58
Descriptor:
     -> Name: kube_node_info
NumberDataPoints #0
     -> node: Str(otel-metadata-demo-control-plane)
     -> kernel_version: Str(...)
     -> os_image: Str(...)
     -> container_runtime_version: Str(containerd://...)
```

What is present: Kubernetes object state — pod-to-node mapping, controller ownership, IP addresses, node system info — all from the Kubernetes API, not from the app or the host.

What is absent: `host.name`, `host.id`, `k8s.pod.uid` (as resource attributes) — this data path never passes through k8sattributes or hostmetrics.

### 5. Summary of what arrives where

The gateway's debug exporter output proves three independent `Resource` blocks:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          Gateway debug output                            │
├──────────────────────┬──────────────────────┬────────────────────────────┤
│ ResourceSpans        │ ResourceMetrics      │ ResourceMetrics            │
│ (from agent traces   │ (from agent metrics  │ (from prometheus receiver  │
│  pipeline)           │  pipeline)           │  scraping KSM)            │
├──────────────────────┼──────────────────────┼────────────────────────────┤
│ service.name         │ host.name            │ service.name (=job/KSM)    │
│ service.version      │                      │ service.instance.id        │
│ telemetry.sdk.*      │                      │ net.host.name              │
│ k8s.namespace.name   │                      │                            │
│ k8s.pod.name         │                      │ Data point attributes:     │
│ k8s.pod.uid          │                      │   namespace                │
│ k8s.deployment.name  │                      │   pod                      │
│ k8s.node.name        │                      │   node                     │
│                      │                      │   created_by_kind          │
│                      │                      │   host_ip, pod_ip          │
├──────────────────────┼──────────────────────┼────────────────────────────┤
│ no host.ip           │ no k8s.pod.name      │ no host.name               │
│ no host.mac          │ no k8s.namespace     │ no k8s.pod.uid (resource)  │
│ no host.id           │ no k8s.deployment    │ no host.id                 │
└──────────────────────┴──────────────────────┴────────────────────────────┘
```

## Components

### trace-generator (Go)

A minimal Go application (`app/main.go`) instrumented with the OTel Go SDK. It exports traces over OTLP gRPC to the agent on the same node (resolved via the `HOST_IP` downward API field). The app:
- Runs a background goroutine that emits a `background-tick` span every 5 seconds
- Serves `GET /generate` which creates a parent span with a child `compute` span
- Serves `GET /health` for liveness checks
- Sets `service.name=trace-generator` and `service.version=0.1.0` as resource attributes

### OTel Agent (DaemonSet)

Runs `otel/opentelemetry-collector-contrib` on every node with `hostNetwork: true` so apps can reach it at `<node-ip>:4317`. Configuration is in `k8s/otel-agent-configmap.yaml`.

Two independent pipelines:

| Pipeline | Receiver | Processors | Exporter | Produces |
|----------|----------|------------|----------|----------|
| `traces` | `otlp` (app spans) | `k8sattributes`, `batch` | `otlp` (to gateway) | ResourceSpans with `k8s.*` |
| `metrics` | `hostmetrics` (cpu, memory, filesystem, network, load) | `batch` | `otlp` (to gateway) | ResourceMetrics with `host.*` |

The `k8sattributes` processor is configured with:
- `filter.node_from_env_var: KUBE_NODE_NAME` — only looks up pods on this node (DaemonSet pattern)
- `pod_association: [from: connection]` — matches telemetry to pods by source IP
- Extracts: `k8s.namespace.name`, `k8s.pod.name`, `k8s.pod.uid`, `k8s.deployment.name`, `k8s.node.name`

The DaemonSet mounts `/proc` and `/sys` from the host for the hostmetrics receiver and has RBAC permissions to list/watch pods, namespaces, nodes, replicasets, and deployments for k8sattributes.

### OTel Gateway (Deployment)

A single-replica Deployment running `otel/opentelemetry-collector-contrib`. Configuration is in `k8s/otel-gateway-configmap.yaml`.

Two receivers feed into two pipelines, all exported to `debug` (stdout with `verbosity: detailed`):

| Pipeline | Receivers | What it carries |
|----------|-----------|-----------------|
| `traces` | `otlp` (forwarded from agent) | App traces with `k8s.*` resource attributes |
| `metrics` | `otlp` (forwarded from agent) + `prometheus` (KSM scrape) | Host metrics with `host.*` attributes AND KSM metrics with object-state labels |

### kube-state-metrics

Deployed from `registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.12.0` with RBAC to watch standard Kubernetes resources. Exposes a Prometheus metrics endpoint on port 8080 that the gateway scrapes.

## Files

```
apps/hostmetadata-challenges/
├── app/
│   ├── main.go              # Trace-generating Go application
│   ├── go.mod               # Go module definition
│   ├── go.sum               # Go dependency checksums
│   └── Dockerfile           # Multi-stage build
├── k8s/
│   ├── namespace.yaml              # otel-multi-pipeline-demo namespace
│   ├── app-deployment.yaml         # trace-generator Deployment + Service
│   ├── otel-agent-configmap.yaml   # Agent collector config (otlp + hostmetrics)
│   ├── otel-agent-daemonset.yaml   # Agent DaemonSet + RBAC
│   ├── otel-gateway-configmap.yaml # Gateway collector config (otlp + prometheus)
│   ├── otel-gateway-deployment.yaml# Gateway Deployment + Service
│   └── kube-state-metrics.yaml     # KSM Deployment + Service + RBAC
├── Makefile                 # cluster, build, deploy, logs, clean targets
└── README.md                # This file
```

## Cleanup

```bash
make clean
```

This deletes the entire kind cluster.

## Troubleshooting

**Agent pod not starting:** The agent uses `hostNetwork: true` and binds ports 4317/4318 on the node. If another process already binds those ports, the agent will fail. Check with `kubectl -n otel-multi-pipeline-demo describe pod -l app=otel-agent`.

**No traces in gateway logs:** The app emits a background span every 5 seconds and the agent batches with a 10-second timeout. Wait about 30 seconds after deploy, then check `make logs-gateway`. If still empty, check that the agent is forwarding: `make logs-agent`.

**kube-state-metrics not being scraped:** Verify the KSM service is reachable from the gateway:
```bash
kubectl -n otel-multi-pipeline-demo port-forward svc/kube-state-metrics 9090:8080
curl http://localhost:9090/metrics | head -20
```

**`host.ip` / `host.mac` not present in host metrics:** Expected in kind. The hostmetrics receiver only sets `host.name`. In production, add a [`resourcedetection` processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor) with cloud-specific detectors to the agent's metrics pipeline.

## Resources

1. [OpenTelemetry Kubernetes Getting Started][1] — DaemonSet + Deployment architecture
2. [k8sattributes processor][2] — Pod metadata enrichment via Kubernetes API
3. [OTel Host Semantic Conventions][3] — `host.id`, `host.ip`, `host.mac` definitions
4. [hostmetrics receiver][4] — System metrics from `/proc` and `/sys`
5. [kube-state-metrics][5] — Kubernetes object-state as Prometheus metrics

[1]: https://opentelemetry.io/docs/platforms/kubernetes/getting-started/
[2]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/processor/k8sattributesprocessor/README.md
[3]: https://opentelemetry.io/docs/specs/semconv/resource/host/
[4]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/hostmetricsreceiver/README.md
[5]: https://github.com/kubernetes/kube-state-metrics
