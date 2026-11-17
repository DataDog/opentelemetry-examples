# Adapt an existing setup to DDOT

Use this guide to adapt existing application instrumentation and Kubernetes telemetry infrastructure to DDOT. It
supports applications with no instrumentation, upstream OpenTelemetry, existing Datadog instrumentation, or a mixture
of these. This path adapts the existing installation without reinstalling it with the [`install` script][install].

The public Datadog tracing documentation owns language-specific SDK installation and activation. This guide owns the
migration decisions that span those SDKs: inventory the existing setup, choose one instrumentation path, move compatible
configuration into environment variables, preserve intentional features, and verify each signal. The Kubernetes rules
then adapt the Collector, OpenTelemetry Operator, and Datadog Agent. The application workflow applies to every
deployment. Apply the remaining rules only when the corresponding Kubernetes component is present.

Each Kubernetes requirement is a rule with an ID, what to check, how to fix it, and how it can conflict with an existing
setup. [`values.yaml`][values] and [`datadog-agent.yaml`][dd-agent-yaml] are the executable reference for this
repository's Kubernetes architecture. They do not replace intentional choices in another valid architecture.

Contents:

- [Adapt applications](#adapt-applications)
- [Instructions for coding agents](#instructions-for-coding-agents)
- [Target architecture](#target-architecture)
- [OpenTelemetry Collector rules](#opentelemetry-collector-rules)
- [OpenTelemetry Operator and `Instrumentation` rules](#opentelemetry-operator-and-instrumentation-rules)
- [DDOT SDK environment rules](#ddot-sdk-environment-rules)
- [Datadog Agent and Datadog Operator rules](#datadog-agent-and-datadog-operator-rules)
- [Known gaps](#known-gaps)
- [Verify and report](#verify-and-report)

## Adapt applications

Apply this workflow to each application before changing its deployment infrastructure.

### 1. Inspect the existing setup

Record the runtime, architecture, package manager, lockfile, resolved SDK versions, process command, and image build.
Find every instrumentation activation mechanism. For traces, metrics, and logs, find code that creates or globally
registers a `TracerProvider`, `MeterProvider`, or `LoggerProvider`, and find code or configuration that selects an
exporter, such as OTLP. Also find every `OTEL_*` and `DD_*` setting, Collector or Datadog Agent destination, and direct
telemetry path. Include configuration in application code, container environments, Compose files, Kubernetes manifests,
ConfigMaps, Secrets, image `ENV` instructions, and startup scripts.

Classify the application and choose its migration path:

| Starting point | Action |
|---|---|
| No instrumentation | Install and activate one DDOT SDK. |
| Upstream OpenTelemetry | Keep API instrumentation. Determine whether application code or auto-instrumentation creates each `TracerProvider`, `MeterProvider`, or `LoggerProvider` and configures its exporter before replacing the upstream OpenTelemetry SDK setup. |
| Datadog instrumentation | Reuse the installed Datadog SDK when its version supports the required OTLP signals. |
| Mixed instrumentation | Resolve components that instrument the same libraries or both register a global `TracerProvider`, `MeterProvider`, or `LoggerProvider` for the same signal before enabling another loader. |

Preserve explicit disables and intentional secondary exporters until the user approves a change. Prevent duplicate
activation, such as two Java agents, two Node.js initialization hooks, a CLR profiler with another injection mechanism,
or a Go binary compiled by two instrumentation tools.

### 2. Install or reuse one SDK

Use the public setup documentation below only to install and activate the selected SDK. Use the
[OTLP export support matrix][dd-otlp-export] to configure its transport. A Datadog Agent is optional unless the
deployment uses one of the Datadog Agent-backed features described in
[Configure destinations and preserve optional features](#4-configure-destinations-and-preserve-optional-features).

| Language | Setup documentation |
|---|---|
| Java | [Java tracing setup][java-setup] |
| Node.js | [Node.js tracing setup][node-setup] |
| Python | [Python tracing setup][python-setup] |
| .NET | [.NET tracing setup][dotnet-setup] |
| Go | [Go tracing setup][go-setup] and [Orchestrion][orchestrion] |

Check [OpenTelemetry API support][dd-otel-api] and the [OTLP export support matrix][dd-otlp-export] for the selected
version. Use the application's package manager and lockfile, pin the resolved version, and verify that exactly one
loader, profiler, runtime agent, or compile-time tool activates instrumentation. For Operator injection, take the
current image reference from `instrumentation.<language>.image` in [`values.yaml`][values].

### 3. Move compatible configuration to environment variables

Use standard `OTEL_*` environment variables for common configuration shared by upstream OpenTelemetry SDKs and DDOT
SDKs, including service identity, resource attributes, exporter destinations, protocols, headers, signal selection,
sampling, and propagation. This keeps deployment configuration under the control of the team operating the application.
Refactor application code only when an environment variable provides equivalent behavior. Keep code that implements
custom processors, readers, exporters, or other behavior that environment variables cannot express. Do not register a
second global `TracerProvider`, `MeterProvider`, or `LoggerProvider` for the same signal.

Keep OpenTelemetry API calls. Apply the language-and-signal-specific [OpenTelemetry API migration
documentation][dd-otel-api] separately to traces, metrics, and logs. The DDOT SDK might register a `TracerProvider` and
configure its exporter while application code still registers a `MeterProvider` or `LoggerProvider`. Remove code that
creates or registers one of these SDK components, its SDK package, or its exporter only when the relevant
language and signal documentation says the DDOT SDK replaces it. If a signal must also reach another backend, send it
once to the Collector and add another Collector exporter when possible. This avoids maintaining two in-process export
pipelines for the same signal. If a custom in-process processor, reader, or exporter has no DDOT or Collector
equivalent, report the migration as blocked. For Operator injection, check whether the injected SDK registers the
`TracerProvider`, `MeterProvider`, or `LoggerProvider` before retaining application bootstrap code for that signal.

Set the service identity explicitly and use the same values for every signal:

```text
OTEL_SERVICE_NAME=<SERVICE_NAME>
OTEL_RESOURCE_ATTRIBUTES=service.namespace=<NAMESPACE>,service.version=<VERSION>,deployment.environment.name=<ENVIRONMENT>
```

When migrating existing Datadog instrumentation to OpenTelemetry mode, migrate `DD_SERVICE`, `DD_ENV`, `DD_VERSION`,
and compatible `DD_TAGS` values to `OTEL_SERVICE_NAME` and `OTEL_RESOURCE_ATTRIBUTES`. Convert Datadog tag syntax to
OpenTelemetry resource-attribute syntax instead of copying the `DD_TAGS` string. Remove overlapping identity keys from
the `DD_*` settings because they can take precedence over the OpenTelemetry values. Keep a `DD_*` identity setting only
while a remaining Datadog-specific feature or native transport requires it, and keep its value aligned with the
canonical OpenTelemetry identity. Verify that both paths produce the same `service`, `env`, and `version`. Merge
existing resource attributes instead of replacing them. See [OpenTelemetry environment variable
interoperability][dd-otel-env] for mappings and precedence.

Enable traces, metrics, and logs by default. Check the selected SDK's support and required packages for each signal,
then verify all three signals independently. When an SDK cannot export a signal, or the application intentionally
registers that signal's `TracerProvider`, `MeterProvider`, or `LoggerProvider` and exporter, record the exception and
keep that signal disabled in the DDOT SDK until the missing support or duplicate registration is resolved. Successful
trace export does not establish metrics or logs support. The Kubernetes reference configuration keeps the complete
shared environment list in `instrumentation.env` in [`values.yaml`][values].

### 4. Configure destinations and preserve optional features

Point each enabled signal at a reachable OTLP endpoint. Use the SDK's documented protocol default when the endpoint
accepts it. Set a shared or signal-specific protocol only when the selected SDK or destination requires an override.
Check the [OTLP export support matrix][dd-otlp-export] when changing an SDK version. That page owns the trace-export
switches and supported transports; [environment variable interoperability][dd-otel-env] owns Datadog/OpenTelemetry
mappings and precedence.

For an SDK configured manually, use one shared endpoint when traces, metrics, and logs go to the same local Collector:

```text
OTEL_EXPORTER_OTLP_ENDPOINT=http://<COLLECTOR_HOST>:4318
```

Use a signal-specific endpoint only when that signal has a different destination. Signal-specific endpoints override
the shared endpoint. For OTLP/HTTP, include the signal path, such as `/v1/traces`, `/v1/metrics`, or `/v1/logs`. For
OTLP/gRPC, use the host and port without an HTTP signal path.

The destination pipeline depends on the deployment. This repository's reference architecture exports from the DDOT
Collector to Datadog with the Datadog exporter. Preserve intentional secondary exporters and custom routing, and verify
that no path sends a second copy of the same signal. Preserve a compatible existing export path instead of replacing it
only to match this repository. For a new or different Collector deployment, follow the [current Datadog Collector
setup][dd-collector-setup].

For Kubernetes, deploy a Datadog Agent alongside DDOT when the application uses Datadog Agent-backed capabilities or
the team wants to enable them later. The basic OTLP path from the SDK to a Collector works without a Datadog Agent.
Features such as Remote Configuration, Live Debugger, Dynamic Instrumentation, and DogStatsD use the Datadog Agent.
Set `DD_AGENT_HOST` when a reachable Datadog Agent provides a feature the application uses. If the SDK's runtime metrics
use DogStatsD, expose UDP `8125` and keep that Datadog Agent path. The Datadog SDK
[runtime-metrics guide][dd-runtime-metrics] owns the language-specific defaults and metric names. If the application
sends OTLP directly to a Datadog Agent, enable its OTLP receiver and use Datadog Agent 7.48.0 or later for logs. Avoid
sending the same signal through both Collector and Datadog Agent paths.

Switch trace export from the native Datadog Agent path to OTLP only after OTLP trace delivery succeeds. Do not disable
the tracer; follow the [OTLP export documentation][dd-otlp-export] for the exporter switch. Review direct log
submission, file tailing, DogStatsD, and runtime metrics independently because they do not all follow the trace
exporter.

## Instructions for coding agents

Run the coding agent from the repository that owns the applications and deployment configuration. Ask it to follow this
guide, preserve intentional choices, report conflicts before changing them, and verify every enabled signal through the
Collector and in Datadog.

1. **Adapt each application.** Record its starting point, SDK activation, whether application code or an injected SDK
   registers the `TracerProvider`, `MeterProvider`, and `LoggerProvider` and configures their exporters, effective
   environment, destinations, and enabled signals.
2. **Find the Kubernetes components** in the repository: OpenTelemetry Collector configurations (Helm values of the
   `opentelemetry-kube-stack` or `opentelemetry-collector` charts, `OpenTelemetryCollector` custom resources, raw
   Collector YAML), `Instrumentation` custom resources, workload manifests with `instrumentation.opentelemetry.io/*`
   annotations or `OTEL_*`/`DD_*` environment variables, `DatadogAgent` custom resources, and Helm values of the
   `datadog` or `datadog-operator` charts. Only apply the rule sections of the components you found.
   For Docker Compose or another non-Kubernetes deployment, find the application commands and environment, Collector
   service and configuration, and any Datadog Agent service, then apply only the application workflow and public
   Collector setup.
3. **Preserve existing configuration.** Keep existing receivers, exporters, pipelines, and environment variables unless a rule
   says otherwise. Follow the rule's **Conflicts** line when settings differ; ask the user when it does not cover the
   conflict.
4. **Account for Helm list replacement.** Repeat every item of an overridden list, including pipeline
   `receivers`/`processors`/`exporters`, `env`, and `operators`.
5. **Use placeholders for unknown values.** List every placeholder in the report. Reference Kubernetes Secrets for API
   and application keys.
6. **Use current reference versions.** Read image versions from [`values.yaml`][values] and
   [`datadog-agent.yaml`][dd-agent-yaml] instead of copying versions from prose.
7. **Report** every rule as `applied`, `already satisfied`, `skipped` with a reason, or `blocked` by a known gap or
   missing value.

## Target architecture

![Deployment architecture: the OpenTelemetry Kube Stack deploys the DDOT SDKs, Collectors, and host profiler, next to a Datadog Agent acting as an enabler of the DDOT SDKs][architecture]

After the changes in this guide:

- **OpenTelemetry Collectors** collect the Kubernetes and application telemetry and export it to Datadog. A
  `daemon` collector (DaemonSet) receives the applications' OTLP traces, metrics, and logs, and collects node-scope
  telemetry, including the telemetry of the middleware running on its node. A `cluster` collector (single-replica Deployment) collects cluster-scope telemetry.
- **The OpenTelemetry Operator** injects the DDOT SDKs into annotated workloads through an `Instrumentation` custom
  resource, and points them at the `daemon` collector.
- **The DDOT SDKs** export OTLP to the `daemon` collector.
- **The Datadog Agent** only acts as an **enabler of the DDOT SDKs**: it enables their Datadog-specific features, such
  as Live Debugger, Dynamic Instrumentation, Remote Configuration of head sampling, or Feature Flag Management. It
  doesn't collect the Kubernetes telemetry or instrument the applications, so its overlapping features are disabled.

## OpenTelemetry Collector rules

These rules apply to the Collector configuration, whatever the way it's deployed. The paths refer to the
`opentelemetry-kube-stack` chart, where `defaultCRConfig` is merged into every collector and `collectors.<name>`
configures one collector.

### C-01: Export to Datadog with the Datadog exporter

- **Requirement:** a `datadog` exporter, with the API key and site read from a Kubernetes Secret.
- **Fix:**

  ```yaml
  env:
    - name: DD_API_KEY
      valueFrom:
        secretKeyRef: { name: datadog-secret, key: api-key }
    - name: DD_SITE
      valueFrom:
        secretKeyRef: { name: datadog-secret, key: dd-site, optional: true }
  config:
    exporters:
      datadog/exporter:
        api:
          key: ${env:DD_API_KEY}
          site: ${env:DD_SITE:-datadoghq.com}
  ```

- **Conflicts:** keep the user's other exporters next to it, for example to export to another observability backend
  too.

### C-02: Compute APM trace metrics with the Datadog connector, before any sampling

- **Requirement:** the traces received from the applications go through the `datadog` connector, which computes the APM
  trace metrics, then to a second traces pipeline that exports them. The connector also feeds the `metrics` pipeline.
- **Why:** the connector must see 100% of the traces for the APM metrics (requests, errors, latency) to be accurate.
- **Fix:**

  ```yaml
  connectors:
    datadog/connector: {}
  service:
    pipelines:
      traces:
        receivers: [otlp]
        exporters: [datadog/connector]
      traces/sampling:              # add sampling processors here, after the connector
        receivers: [datadog/connector]
        exporters: [datadog/exporter]
      metrics:
        receivers: [datadog/connector, otlp]
        exporters: [datadog/exporter]
  ```

- **Conflicts:** move any existing `tail_sampling` or `probabilistic_sampler` processor from the `traces` pipeline to
  `traces/sampling`. Exporters to other observability backends can stay in the `traces` pipeline.

### C-03: Enrich the telemetry with the resource attributes Datadog needs

- **Requirement:** the traces, metrics, and logs of a workload carry the same values of the following resource
  attributes: Datadog correlates the telemetry on them, and can't link the signals of a workload when they differ.
  `service.namespace` is strongly recommended rather than required.

  | Resource attribute                                                                                                                                                                                                                  | Datadog use                                                                                           | Set by, in the reference configuration                                                                                                                                   |
  |-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
  | `service.name`                                                                                                                                                                                                                      | `service` tag, needed for any correlation                                                             | OpenTelemetry Operator and `k8s_attributes`, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]: annotations, labels, workload names      |
  | `service.namespace`                                                                                                                                                                                                                 | Groups the services of an application. Strongly recommended, not required                             | OpenTelemetry Operator and `k8s_attributes`, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]: annotations, Kubernetes namespace        |
  | `deployment.environment.name`                                                                                                                                                                                                       | `env` tag                                                                                             | OpenTelemetry Operator and `resourceAttributes` of the kube-stack chart ([C-11](#c-11-set-the-deployment-environment)): annotations, OTel Collector pipeline             |
  | `service.version`                                                                                                                                                                                                                   | `version` tag                                                                                         | OpenTelemetry Operator and `k8s_attributes`, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]: annotations, labels, container image tag |
  | `container.id`                                                                                                                                                                                                                      | Links spans to the container metrics: without it, APM doesn't show the container metrics of a service | The SDK's container resource detector: not extracted by `k8s_attributes` in the reference                                                                                |
  | `host.name`                                                                                                                                                                                                                         | Links to the host metrics                                                                             | `resource/hostname`, from `k8s.node.name`                                                                                                                                |
  | `host.id`, `cloud.provider`, `cloud.account.id`                                                                                                                                                                                     | Datadog hostname of cloud hosts, cloud context                                                        | `resource_detection` with the cloud provider's detector                                                                                                                  |
  | `datadog.host.name`                                                                                                                                                                                                                 | Datadog hostname, when `host.id` can't be reliably discovered                                         | Not set in the reference as `k8s.cluster.name` is set when it can't be discovered                                                                                        |
  | `k8s.pod.name`                                                                                                                                                                                                                      | Links the pod and container views                                                                     | OpenTelemetry Operator and OpenTelemetry Operator, `k8s_attributes`                                                                                                      |
  | `k8s.container.name`                                                                                                                                                                                                                | Container name displayed in Datadog                                                                   | OpenTelemetry Operator (`OTEL_RESOURCE_ATTRIBUTES`), `k8s_attributes` for the other workloads                                                                            |
  | `k8s.cluster.name`                                                                                                                                                                                                                  | Kubernetes Explorer, Kubernetes dashboards                                                            | `resource_detection` (EKS, GKE, AKS) or `resourceAttributes` ([C-10](#c-10-use-the-same-kubernetes-cluster-name-everywhere))                                             |
  | `k8s.namespace.name`, `k8s.node.name`, `k8s.pod.uid`, `k8s.deployment.name`, `k8s.replicaset.name`, `k8s.daemonset.name`, `k8s.statefulset.name`, `k8s.job.name`, `k8s.cronjob.name`, `container.image.name`, `container.image.tag` | Kubernetes context of the telemetry                                                                   | `k8s_attributes`                                                                                                                                                         |

- **Fix:** enrich the telemetry in the Collector with the `k8s_attributes` processor, in every pipeline that receives
  telemetry from the workloads, rather than setting these attributes by hand in each application:
  - with the kube-stack chart, `presets.kubernetesAttributes.enabled: true`. It extracts the attributes above, except
    `container.id` and `k8s.cluster.name`, and sets `otel_annotations: true`;
  - otherwise, the `k8s_attributes` processor with its RBAC (`get`, `list`, and `watch` on pods, namespaces, nodes, and
    replica sets), and the `metadata` to extract:

    ```yaml
    processors:
      k8s_attributes:
        passthrough: false
        filter:
          node_from_env_var: K8S_NODE_NAME   # on a DaemonSet: only watch the pods of its node
        extract:
          otel_annotations: true             # strongly recommended: resource.opentelemetry.io/<attribute> annotations
          metadata:
            - k8s.namespace.name
            - k8s.pod.name
            - k8s.pod.uid
            - k8s.node.name
            - k8s.deployment.name
            - k8s.replicaset.name
            - k8s.daemonset.name
            - k8s.statefulset.name
            - k8s.job.name
            - k8s.cronjob.name
            - k8s.container.name
            - container.id
            - container.image.name
            - container.image.tag
            - service.name
            - service.namespace
            - service.version
            - service.instance.id
        pod_association:
          - sources: [{ from: resource_attribute, name: k8s.pod.uid }]
          - sources: [{ from: resource_attribute, name: k8s.pod.ip }]
          - sources: [{ from: connection }]
    ```

- **Recommendations for the resource attributes on Kubernetes:**
  - **All attributes:** follow the OpenTelemetry [Specify resource attributes using Kubernetes
    annotations][otel-semconv-k8s-attributes] conventions, which `k8s_attributes` and the OpenTelemetry Operator
    implement.
  - **`k8s_attributes`:** strongly recommended, set `extract.otel_annotations: true`, so that the processor adds the
    `resource.opentelemetry.io/<attribute>` pod annotations to all the telemetry of the pod, including the container
    metrics and logs that the Collectors collect.
  - **`service.name` and `service.namespace`:** set them with the `resource.opentelemetry.io/service.name` and
    `resource.opentelemetry.io/service.namespace` annotations of the pod template:

    ```yaml
    spec:
      template:
        metadata:
          annotations:
            resource.opentelemetry.io/service.name: my-service
            resource.opentelemetry.io/service.namespace: my-application
    ```

  - **`service.version`:** rely on the [OpenTelemetry conventions][otel-semconv-k8s-attributes], which derive it from
    the container image tag: tag the images with the application version. The `app.kubernetes.io/version` pod label
    and the `resource.opentelemetry.io/service.version` pod annotation also work, but require a more sophisticated
    software delivery pipeline, which updates them on every release.
  - **`deployment.environment.name`:** don't set it in the application's deployment manifests, so that the same
    deployment pipeline and manifests serve all the environments. Set it in the Collector pipeline instead, typically
    with the `resourceAttributes` of the kube-stack chart ([C-11](#c-11-set-the-deployment-environment)).
  - **`k8s.node.name`:** set it with the `k8s_attributes` processor, from the pod the telemetry comes from.
  - **`host.id`, `cloud.provider`, and `cloud.account.id`:** set them whenever the Collector can discover them,
    typically with the `resource_detection` processor and the cloud provider's detector (`eks`, `aks`, `gke`...), so
    that Datadog attaches the telemetry to the right host
    ([hostname mapping][dd-hostname-mapping]). Check that the detector enables these attributes in its
    `resource_attributes`, and only set `host.id` on the telemetry of the Collector's own node, that is, in the
    `daemon` collector behind a node-local Service ([C-13](#c-13-keep-the-daemon-collectors-service-node-local)): not
    on the cluster-scope telemetry of the `cluster` collector, which describes other nodes.
  - **On EKS:** enable these resource attributes of the `eks` detector, because several of them are disabled by
    default. The kube-stack preset (`presets.resourceDetection.eks.enabled: true`) only enables the detector, so the
    [`install`][install] script also sets these attributes with `--set` flags on `defaultCRConfig` when the cluster is
    EKS:

    ```yaml
    processors:
      resource_detection/env:
        eks:
          node_from_env_var: K8S_NODE_NAME   # the node to read the EC2 instance attributes from
          resource_attributes:
            host.id:
              enabled: true
            cloud.region:
              enabled: true
            cloud.availability_zone:
              enabled: true
            cloud.account.id:
              enabled: true
            host.image.id:
              enabled: true
            host.type:
              enabled: true
    ```
  - **`datadog.host.name`:** don't set it where `k8s.cluster.name`, `k8s.node.name`, `host.id`, and the `cloud.*`
    attributes can be reliably discovered, such as on EKS, AKS, and GKE: Datadog then derives the hostname from them.
    Only set `datadog.host.name` when those can't be reliably discovered, for example on non-cloud Kubernetes clusters, set
    `datadog.host.name` to `<node name>-<cluster name>` in the `daemon` collector, the same value as the Datadog
    Agent's hostname ([A-06](#a-06-use-the-same-kubernetes-cluster-name-as-the-opentelemetry-collectors)), so that the
    Datadog Agent and the Collectors report the same host:

    ```yaml
    env:
      - name: K8S_NODE_NAME
        valueFrom:
          fieldRef: { fieldPath: spec.nodeName }
      - name: K8S_CLUSTER_NAME
        value: "<CLUSTER_NAME>"
    config:
      processors:
        resource/datadog_hostname:
          attributes:
            # only set datadog.host.name when k8s.*, host.id, and cloud.* can't be reliably discovered
            - action: upsert
              key: datadog.host.name
              value: ${env:K8S_NODE_NAME}-${env:K8S_CLUSTER_NAME}
    ```

- **`k8s.container.name`:** extract it with the `k8s_attributes` processor, in its `extract.metadata`, as the
  kube-stack preset does. The OpenTelemetry Operator injects it in `OTEL_RESOURCE_ATTRIBUTES` only for the workloads it
  instruments: `k8s_attributes` sets it for the other ones, such as applications instrumented without the Operator.
  In pods with several containers, the processor can only tell which container sent the telemetry from its
  `container.id` resource attribute, and doesn't set `k8s.container.name` without it.
- **`container.id`:** check that the application traces carry it. If they don't, add `container.id` to the
  `k8s_attributes` processor's `extract.metadata`: the processor then looks up the container ID from the pod's status,
  using the `k8s.container.name` that the OpenTelemetry Operator injects in `OTEL_RESOURCE_ATTRIBUTES`.
- **Conflicts:** `k8s_attributes` doesn't overwrite attributes the telemetry already has. Remove the hand-set values
  that differ from the Kubernetes metadata, such as a `host.name` set to the container's hostname by the SDK's host
  detector, rather than keeping two sources of truth.
- **Docs:** [Correlate OpenTelemetry traces and metrics][dd-correlate-metrics-traces],
  [correlate traces with container metrics][dd-docker-metrics-correlate],
  [Monitor Kubernetes with OpenTelemetry][dd-k8s-otel], the [Collector's Kubernetes manifest
  reference][dd-collector-k8s-manifest], and [Unified Service Tagging][dd-unified-service-tagging].

FIXME verify that the Datadog OTel Contrib exporter and extension can discover `datadog.host.name` passed as an
environment variable, typically through `OTEL_RESOURCE_ATTRIBUTES=datadog.host.name=<node name>-<cluster name>`.

### C-04: Don't stamp the Collector's own identity on incoming telemetry

- **Requirement:** no `env` detector of the `resource_detection` processor on collectors that receive application,
  middleware, or host telemetry, and `override: false` on the `resource_detection` processors.
- **Why:** the `env` detector adds the Collector pod's own `k8s.pod.name`, `k8s.namespace.name`, `node.name`, and
  `host.name` to the telemetry it receives, overwriting the application's identity.
- **Fix:** with the kube-stack chart, `collectors.daemon.presets.resourceDetection.env.enabled: false` (enabled by the
  chart's daemon defaults), and:

  ```yaml
  processors:
    resource_detection/env:
      override: false
  ```

- **Conflicts:** keep the cloud provider detectors (EKS, GKE, AKS) and `k8s_api`: they detect `k8s.cluster.name`.

### C-06: Convert cumulative metrics to delta

- **Requirement:** the `cumulative_to_delta` processor in the metrics pipelines receiving cumulative metrics (OTLP,
  Prometheus receivers), and `delta_to_rate` for the counters that Datadog displays as rates.
- **Reference:** `defaultCRConfig.config.processors` and `collectors.daemon.config.processors` in [`values.yaml`][values].
- **Conflicts:** applications should also export delta metrics, see [S-04](#s-04-export-delta-metrics).
- **Conflicts:** when a metrics pipeline also exports to destinations that require cumulative temporality, such as
  Prometheus remote write or another observability backend, don't convert to delta in that shared pipeline: split it
  with a `forward` connector into a fan-out with two ends. The cumulative end exports without conversion, and the
  Datadog end adds `cumulative_to_delta` and `delta_to_rate` before the Datadog exporter. Keep the shared processors
  (`k8s_attributes`, `resource_detection`) in the upstream pipeline, and the `datadog/connector`
  as a receiver of the Datadog end only:

  ```yaml
  connectors:
    forward/metrics: {}
  service:
    pipelines:
      metrics:                         # shared: receive and enrich once
        receivers: [otlp]
        processors: [k8s_attributes, resource_detection/env]
        exporters: [forward/metrics]
      metrics/cumulative:              # destinations requiring cumulative temporality
        receivers: [forward/metrics]
        exporters: [prometheusremotewrite/other-backend]
      metrics/datadog:                 # Datadog requires delta temporality
        receivers: [forward/metrics, datadog/connector]
        processors: [cumulative_to_delta, delta_to_rate]
        exporters: [datadog/exporter]
  ```

  In this case, the applications must export cumulative metrics, not delta ones as
  [S-04](#s-04-export-delta-metrics) requires, so that the cumulative end receives cumulative metrics.

### C-07: Report the Collectors to Fleet Automation with the Datadog extension

- **Requirement:** the `datadog` extension, with `deployment_type: gateway` on Deployments and
  `deployment_type: daemonset` on DaemonSets, listed in `service.extensions`.
- **Why:** it lists the Collectors in Datadog [Fleet Automation][dd-fleet-automation], with their version and
  configuration.
- **Fix:**

  ```yaml
  extensions:
    datadog:
      api:
        key: ${env:DD_API_KEY}
        site: ${env:DD_SITE:-datadoghq.com}
      deployment_type: daemonset
  service:
    extensions: [health_check, datadog]
  ```

### C-08: Export the Collector's internal metrics through its own pipeline

- **Requirement:** the Collector exports its internal metrics over OTLP to its own OTLP receiver, so that they get the
  same resource attributes and tags as the rest of the telemetry.
- **Why:** the internal metrics then go through the Collector's processors, including the `resource_detection`
  processor of the Kubernetes platform (EKS, AKS, GKE), and get enriched with resource attributes such as `host.id` and
  `k8s.cluster.name`. Exported directly to Datadog, they would miss these attributes, and the Collector's health
  metrics couldn't be correlated with its Kubernetes cluster and host.
- **Fix:**

  ```yaml
  service:
    telemetry:
      metrics:
        readers:
          - periodic:
              exporter:
                otlp:
                  protocol: http/protobuf
                  endpoint: http://localhost:4318
  ```

- **Conflicts:** don't do the same for the Collector's own logs when the `daemon` collector's `filelog` receiver already
  tails the container logs: it would duplicate them.

### C-09: Receive OTLP over both gRPC and HTTP

- **Requirement:** the `otlp` receiver listens on `0.0.0.0:4317` (gRPC) and `0.0.0.0:4318` (HTTP).

### C-10: Use the same Kubernetes cluster name everywhere

- **Requirement:** the `k8s.cluster.name` resource attribute of the Collectors equals the Datadog Agent's
  `clusterName` ([A-06](#a-06-use-the-same-kubernetes-cluster-name-as-the-opentelemetry-collectors)).
- **Fix:** on EKS, GKE, and AKS, enable the cloud provider's resource detection preset, which discovers it.
- **Fix:** on other platforms, set `k8s.cluster.name` both on the telemetry and on the Collector's own telemetry. The
  Collector's Datadog extension ([C-07](#c-07-report-the-collectors-to-fleet-automation-with-the-datadog-extension))
  reports the Collector under the cluster name of its own telemetry in Fleet Automation. With the kube-stack chart,
  `resourceAttributes.k8s.cluster.name` in the values does both:

  ```yaml
  resourceAttributes:
    k8s.cluster.name: <CLUSTER_NAME>
  ```

  Otherwise, add a `resource` processor that upserts `k8s.cluster.name` to the pipelines, and set `k8s.cluster.name`
  under `service.telemetry.resource`, as the [Datadog extension documentation][dd-otel-extension-resource] describes.
- **Constraints:** dot-separated tokens of lowercase letters, digits, and `-`, starting with a letter and ending with a
  letter or a digit, at most 80 characters. See [Kubernetes cluster name constraints][cluster-name].

### C-11: Set the deployment environment

- **Requirement:** `deployment.environment.name` on all the telemetry, for example with `resourceAttributes` in the
  kube-stack chart's values.

### C-12: Collect middleware telemetry from the `daemon` collector on the same node

- **Requirement:** collect the telemetry of the middleware running in the Kubernetes cluster, such as database
  servers, message brokers, or caches, with the `daemon` collector running on the same Kubernetes node as the observed
  pod. Not with the `cluster` collector, a standalone Collector Deployment, or a `daemon` collector on another node.
- **Why:** the Collector enriches the telemetry it collects with the resource attributes of the observed pod, such as
  its `k8s.node.name`. Collected from another node, or through a Kubernetes Service that load balances across pods,
  the telemetry gets the wrong resource attributes, for example the Collector's node or another replica's
  `k8s.pod.name`, and is attached to the wrong host in Datadog.
- **Fix:** in the `daemon` collector, discover the middleware pods of its own node with the `k8s_observer` extension,
  restricted to the node with `node: ${env:K8S_NODE_NAME}`, and create the receivers with `receiver_creator`, which
  adds the discovered pod's `k8s.pod.uid`, `k8s.pod.name`, and `k8s.namespace.name` to the telemetry. For example, for
  PostgreSQL:

  ```yaml
  extensions:
    k8s_observer:
      auth_type: serviceAccount
      node: ${env:K8S_NODE_NAME}   # only the pods of this collector's node
      observe_pods: true
  receivers:
    receiver_creator/middleware:
      watch_observers: [k8s_observer]
      receivers:
        postgresql:
          rule: type == "port" && port == 5432 && pod.labels["app.kubernetes.io/name"] == "postgresql"
          config:
            username: <POSTGRESQL_MONITORING_USER>
            password: ${env:POSTGRESQL_MONITORING_PASSWORD}
            tls:
              insecure: true
  service:
    extensions: [health_check, datadog, k8s_observer]
    pipelines:
      metrics/middleware:
        receivers: [receiver_creator/middleware]
        processors: [k8s_attributes, resource_detection/env, cumulative_to_delta]
        exporters: [datadog/exporter]
  ```

  For Prometheus scrapes of middleware exporters, use `kubernetes_sd_configs` with `role: pod`, restricted to the node
  with a `spec.nodeName=${env:K8S_NODE_NAME}` field selector, instead of `static_configs` targeting a Service, and
  delete the `service.name` and `service.instance.id` resource attributes the Prometheus receiver sets, as for
  `prometheus/cadvisor` in [`values.yaml`][values].
- **Conflicts:** move existing middleware receivers from the `cluster` collector or a standalone Collector Deployment
  to the `daemon` collector. Keep in the `cluster` collector the cluster-scope telemetry (kube-state-metrics,
  Kubernetes objects) and the middleware running outside the Kubernetes cluster, such as managed cloud databases,
  which has no Kubernetes node: don't set `host.name` on its telemetry from the Collector's node.

### C-13: Keep the `daemon` collector's Service node-local

- **Requirement:** the `daemon` collector's Service, the one the SDKs export to
  ([O-04](#o-04-point-the-sdks-at-the-daemon-collector)), has `internalTrafficPolicy: Local`, so that each pod's
  telemetry always reaches the `daemon` collector running on its own Kubernetes node.
- **Why:** with `internalTrafficPolicy: Cluster`, the Service load balances each connection across the `daemon`
  collectors of all the nodes:
  - the points of a time series can reach different Collector instances, which breaks the stateful `cumulative_to_delta`
    conversion ([S-04](#s-04-export-delta-metrics));
  - the telemetry crosses nodes, and availability zones, for nothing;
  - the load of a `daemon` collector no longer depends on the workloads of its node, so its resource allocation no
    longer matches it.
- **Detect:**

  ```sh
  kubectl get service <helm-release-name>-daemon-collector --namespace <namespace> \
    -o jsonpath='{.spec.internalTrafficPolicy}'   # expected: Local
  ```

- **Fix:** the OpenTelemetry Operator already sets `internalTrafficPolicy: Local` on the Services of the collectors in
  `daemonset` mode, and so does the `opentelemetry-collector` Helm chart in `daemonset` mode, unless
  `service.internalTrafficPolicy` is set: remove any explicit `internalTrafficPolicy: Cluster` override. For a Collector
  DaemonSet deployed with plain manifests, set it on its Service:

  ```yaml
  apiVersion: v1
  kind: Service
  spec:
    internalTrafficPolicy: Local
  ```

- **Conflicts:** with `Local`, there's no fallback to another node: the telemetry of a pod running on a node without a
  ready `daemon` collector is dropped. Make the `daemon` collector run on every node running instrumented workloads,
  with the `tolerations` and `nodeSelector` matching the nodes' taints and labels. Don't apply this rule to the
  `cluster` collector, a Deployment.

## OpenTelemetry Operator and `Instrumentation` rules

Apply this section only to workloads instrumented through OpenTelemetry Operator injection. For an SDK installed and
configured manually, follow [Install or reuse one SDK](#2-install-or-reuse-one-sdk) and put the resulting environment
variables in the workload container instead of an `Instrumentation` custom resource.

### O-01: Use the DDOT SDK images in the `Instrumentation` custom resource

- **Requirement:** the `Instrumentation` custom resource uses the DDOT SDK images instead of the upstream OpenTelemetry
  ones. Take the current tags from `instrumentation.<language>.image` in [`values.yaml`][values]:

  | Language | Upstream image                          | DDOT SDK image                                    |
  |----------|-----------------------------------------|---------------------------------------------------|
  | .NET     | `otel/autoinstrumentation-dotnet`       | `datadog/dd-otel-autoinstrumentation-dotnet`      |
  | Java     | `otel/autoinstrumentation-java`         | `datadog/dd-otel-autoinstrumentation-java`        |
  | Node.js  | `otel/autoinstrumentation-nodejs`       | `datadog/dd-otel-autoinstrumentation-js`          |
  | Python   | `otel/autoinstrumentation-python`       | `datadog/dd-otel-autoinstrumentation-python`      |

- **Conflicts:** to migrate workloads one at a time, create a second `Instrumentation` custom resource with the DDOT
  SDK images instead of modifying the existing one, and point each migrated workload's annotation at it
  (`<namespace>/<name>`).

### O-02: Configure the .NET CLR profiler

- **Requirement:** the OpenTelemetry Operator sets `CORECLR_PROFILER` to the upstream OpenTelemetry profiler's CLSID,
  so the `dotnet` section must override it with the Datadog profiler's CLSID:

  ```yaml
  dotnet:
    env:
      - name: CORECLR_PROFILER
        value: "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}"
      - name: CORECLR_PROFILER_PATH
        value: /otel-auto-instrumentation-dotnet/linux/Datadog.Trace.ClrProfiler.Native.so
      # On musl/Alpine application containers, use the linux-musl subdirectory instead:
      # value: /otel-auto-instrumentation-dotnet/linux-musl/Datadog.Trace.ClrProfiler.Native.so
      - name: DD_DOTNET_TRACER_HOME
        value: /otel-auto-instrumentation-dotnet
  ```

### O-03: Size the Python auto-instrumentation volume

- **Requirement:** for Operator-managed Python injection, use the current DDOT SDK for Python image from
  [`values.yaml`][values]. When selecting another image, verify that its extracted contents fit the
  auto-instrumentation volume configured by the installed OpenTelemetry Operator before deployment. If they do not,
  choose a compatible image or configure the volume using the field supported by that Operator version; do not copy a
  deprecated volume field from an older example.
- **Alpine/musl workloads:** add the pod-template annotation
  `instrumentation.opentelemetry.io/otel-python-platform: "musl"` (glibc is the default). See
  [Auto-instrumenting a Python musl based container](https://opentelemetry.io/docs/platforms/kubernetes/operator/automatic/#annotations-python-musl).

### O-04: Point the SDKs at the `daemon` collector

- **Requirement:** `spec.exporter.endpoint` is the `daemon` collector Service's OTLP HTTP endpoint,
  `http://<helm-release-name>-daemon-collector.<namespace>.svc.cluster.local:4318`. This Service must be node-local
  ([C-13](#c-13-keep-the-daemon-collectors-service-node-local)).
- **Requirement:** `spec.sampler.type: parentbased_always_on` and `spec.propagators: [tracecontext, baggage]`. Sample
  in the Collector instead ([C-02](#c-02-compute-apm-trace-metrics-with-the-datadog-connector-before-any-sampling)).
- **Conflicts:** if the user samples in the SDKs on purpose, keep their sampler and say so in the report: the APM trace
  metrics then only count the sampled traces.

### O-05: Annotate workloads, and prevent double instrumentation

- **Requirement:** each workload's pod template has the injection annotation and the
  `admission.datadoghq.com/enabled: "false"` label:

  ```yaml
  spec:
    template:
      metadata:
        annotations:
          # <language>: dotnet, java, nodejs, python. Use inject-sdk for Go and for SDKs bundled in the image.
          instrumentation.opentelemetry.io/inject-<language>: <namespace>/<instrumentation-name>
        labels:
          admission.datadoghq.com/enabled: "false" # label values are strings: keep the quotes
  ```

- **Why:** the label stops the Datadog Agent's admission controller from also instrumenting the pod. It must be a
  label: the Datadog admission controller only reads pod labels, so a `podSelector` keyed on the OpenTelemetry
  annotations never matches.

### O-06: Instrument Go applications at compile time

- **Requirement:** Go applications are instrumented at build time with [Orchestrion][orchestrion] for the DDOT SDK for
  Go, and use the `instrumentation.opentelemetry.io/inject-sdk` annotation, which only injects the `OTEL_*`
  configuration.
- See also the [known gaps](#known-gaps) for Go.

## DDOT SDK environment rules

The [application workflow](#adapt-applications) owns DDOT SDK installation, activation, signal configuration,
destinations, compatibility, and verification. Put its resulting environment variables in the `Instrumentation`
custom resource's `spec.env` when the Operator configures the workload. Put them in the workload container's
environment when the SDK is installed and configured manually.

### S-01: Enable the OpenTelemetry mode of the DDOT SDKs

- **Requirement:** apply the DDOT OpenTelemetry-mode switches from `instrumentation.env` in
  [`values.yaml`][values] to workloads adopting DDOT SDKs with OpenTelemetry semantic conventions and OTLP export.
  Verify the effective configuration at runtime; presence in `spec.env` is insufficient.

  Merge this fragment into the workload environment. The full shared environment configuration is under
  `instrumentation.env` in [`values.yaml`][values].

  ```yaml
  - name: OTEL_TRACES_EXPORTER
    value: otlp
  - name: DD_TRACE_OTEL_ENABLED
    value: "true"
  - name: DD_LOGS_OTEL_ENABLED
    value: "true"
  - name: DD_METRICS_OTEL_ENABLED
    value: "true"
  - name: DD_TRACE_OTEL_SEMANTICS_ENABLED
    value: "true"
  ```

- **Why:** together, these settings enable OpenTelemetry semantics and OTLP export for the signals supported by the
  selected SDK. `DD_TRACE_OTEL_ENABLED=true` alone does not switch trace export from the Datadog protocol to OTLP.
- **Fix:** merge the variables from `instrumentation.env` in [`values.yaml`][values], preserving other list entries.
  Enable traces, metrics, and logs. Install the packages required by the selected SDK for those signals.
- **Conflicts:** preserve explicit disables, `TracerProvider`, `MeterProvider`, or `LoggerProvider` registrations in
  application code, and alternative exporters until the user approves the migration. Record a signal-specific exception
  when the selected SDK cannot export that signal. Use the [OpenTelemetry API support][dd-otel-api] and
  [OTLP export support matrix][dd-otlp-export] for the selected SDK rather than copying version-specific behavior into
  this guide.

### S-02: Point optional Datadog features at the Datadog Agent

- **Requirement:** for Kubernetes, deploy a Datadog Agent alongside DDOT when the workload uses Datadog Agent-backed
  features or the team wants to enable them later. Set `DD_AGENT_HOST` to that reachable Datadog Agent when a feature
  needs it, such as Remote Configuration, Live Debugger, Dynamic Instrumentation, or DogStatsD runtime metrics.
- **Without a Datadog Agent:** the basic OTLP path from the SDK to a Collector remains available. Preserve an existing
  Datadog Agent destination while a feature still uses it, and avoid sending the same signal through both Collector and
  Datadog Agent paths.

### S-03: Use a compatible OTLP transport

- **Requirement:** point each enabled signal at the `daemon` collector endpoint from
  [O-04](#o-04-point-the-sdks-at-the-daemon-collector). Use the SDK's documented protocol default when the endpoint
  accepts it. Set a shared or signal-specific protocol only when the selected SDK or destination requires an override.
- **Conflicts:** inspect signal-specific endpoint and protocol settings as well as the shared settings. Change a gRPC
  endpoint on port `4317` to HTTP on port `4318` only together with a compatible HTTP protocol. Check the current
  [OTLP export support matrix][dd-otlp-export].

### S-04: Export delta metrics

- **Requirement:** keep the delta temporality preference from `instrumentation.env` in [`values.yaml`][values].

  ```yaml
  - name: OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE
    value: delta
  ```

- **Why:** the SDKs then export delta metrics, which the Collector forwards to Datadog without the memory and CPU cost
  of cumulative-to-delta conversion.
- **Conflicts:** when another destination requires cumulative temporality, export cumulative metrics and convert only
  the Datadog branch with the Collector's `cumulative_to_delta` processor
  ([C-06](#c-06-convert-cumulative-metrics-to-delta)). Size the Collector for the processor's state, set an appropriate
  `max_staleness`, and ensure every point in a time series reaches the same Collector instance through the node-local
  Service ([C-13](#c-13-keep-the-daemon-collectors-service-node-local)).

### S-05: Use the stable semantic conventions

- **Requirement:** keep the stable semantic-convention opt-ins from `instrumentation.env` in
  [`values.yaml`][values].

  ```yaml
  - name: OTEL_SEMCONV_STABILITY_OPT_IN
    value: http,database,rpc,messaging
  ```

### S-06: Enable optional Datadog features per workload

These features are disabled by default. Enable them in the `Instrumentation` resource's `env` for every workload, or in
a workload's container `env` for that workload only:

| Feature | Variables |
|---|---|
| [Database Monitoring][dd-dbm] | `DD_DBM_PROPAGATION_MODE=full`, `DD_DBM_TRACE_PREPARED_STATEMENTS=true` |
| [Data Streams Monitoring][dd-dsm] | `DD_DATA_STREAMS_ENABLED=true`, `DD_TRACE_REMOVE_INTEGRATION_SERVICE_NAMES_ENABLED=true` |
| [App & API Protection][dd-aap] | `DD_APPSEC_ENABLED=true` |
| [Code Security IAST][dd-iast] | `DD_IAST_ENABLED=true` |
| [Code Security SCA][dd-sca] | `DD_APPSEC_SCA_ENABLED=true` |

- **Conflicts:** if the user's workloads already set any of these, keep their values.
- **Precedence:** before changing a variable, find every definition for the workload. The effective order, from highest
  to lowest precedence, is:
  1. the container's `env` in the pod manifest;
  2. the language section of the `Instrumentation` resource, such as `spec.java.env`, for `inject-<language>`;
  3. the `Instrumentation` resource's `spec.env`;
  4. the container's `envFrom` ConfigMaps and Secrets;
  5. the image's `Dockerfile` `ENV` instructions.

  The Operator appends its agent to `JAVA_TOOL_OPTIONS`, `NODE_OPTIONS`, and `PYTHONPATH`, and merges resource
  attributes into `OTEL_RESOURCE_ATTRIBUTES`. It skips a language SDK injection when the container sets that language's
  loader variable with `valueFrom`. See the Operator's [SDK injection implementation][otel-operator-sdk-go].

  After changing an `Instrumentation` resource, restart its workloads and inspect the running process environment:

  ```sh
  kubectl exec <pod> --namespace <namespace> -- env | grep -E '^(DD|OTEL)_'
  ```

  The pod spec does not show values inherited from the image or `envFrom`. Treat runtime output as sensitive because
  exporter headers can contain credentials. Record only names and non-secret effective values in the validation report.

### S-07: Enable runtime metrics where supported

- **Recommendation:** enable runtime metrics for supported workloads to help troubleshoot memory usage, garbage
  collection, and thread activity. Configure this per workload; shared instrumentation defaults remain unchanged.
- **Fix:** set `DD_RUNTIME_METRICS_ENABLED=true` in the workload's container `env`. For OTLP export, keep
  `DD_METRICS_OTEL_ENABLED=true` and `OTEL_METRICS_EXPORTER=otlp` effective for that workload. Check the installed SDK's
  runtime support, required packages or imports, and export transport before applying the change.
- **Compatibility:** follow the [runtime metrics documentation][dd-runtime-metrics]. If the SDK sends runtime metrics
  only through DogStatsD, keep the existing reachable Datadog Agent path. For a setup without a Datadog Agent, report
  runtime-metrics delivery as `blocked` until a supported delivery path is configured.
- **Verify:** confirm that runtime metrics arrive in Datadog with the expected service and environment tags through the
  intended transport. An enabled flag or successful trace export is insufficient.
- **Conflicts:** preserve explicit runtime-metrics disables and existing runtime instrumentation. Ask before enabling a
  disabled collector or adding another one that could duplicate the same measurements.

## Datadog Agent and Datadog Operator rules

These rules use the Datadog Operator's `DatadogAgent` custom resource (`datadoghq.com/v2alpha1`), as in
[`datadog-agent.yaml`][dd-agent-yaml]. With the `datadog` Helm chart, apply the same changes with the chart's equivalent
values: this guide doesn't list them, and they aren't tested in this repository.

### A-01: Disable the features that duplicate the OpenTelemetry Collectors

- **Requirement:** disable a Datadog Agent feature only when its equivalent is enabled in the OpenTelemetry Collectors,
  to avoid collecting the same telemetry twice. The reference [`values.yaml`][values] enables all these equivalents
  except Kubernetes events, but the user may have changed the installer's defaults or configured Collectors without
  some of them: check each equivalent in the user's Collector configuration before disabling the Datadog Agent feature,
  and keep the Datadog Agent feature when its equivalent isn't enabled. The presets add their receivers to the
  pipelines: check the receivers in the rendered `OpenTelemetryCollector` custom resources, not only in the values.

  | Datadog Agent feature                                                                  | OpenTelemetry Collector equivalent, in the `opentelemetry-kube-stack` chart                                                                                                                                          |
  |----------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
  | `kubeStateMetricsCore`, and the legacy `kubernetes_state` check (`DD_IGNORE_AUTOCONF`) | `cluster` collector scraping kube-state-metrics (`prometheus/ksm` receiver), and its `clusterMetrics` preset (`k8s_cluster` receiver)                                                                                |
  | `orchestratorExplorer`, `DD_ORCHESTRATOR_EXPLORER_KUBELET_CONFIG_CHECK_ENABLED`        | `cluster` collector's `kubernetesObjects` preset (`k8s_objects` receiver), and `orchestrator_explorer.enabled: true` on its Datadog exporter                                                                         |
  | `eventCollection.collectKubernetesEvents`                                              | `kubernetesEvents` preset (`k8s_objects` receiver watching the `events.k8s.io` events), on one collector only. The `kubernetesObjects` preset doesn't collect events. **Not enabled in the reference `values.yaml`**       |
  | `logCollection`                                                                        | `daemon` collector's `logsCollection` preset (`file_log` receiver), tailing the container logs, enabled by the chart's `daemon` defaults                                                                             |
  | `clusterChecks`                                                                        | Collector receivers for the cluster-level services, such as the middleware of [C-12](#c-12-collect-middleware-telemetry-from-the-daemon-collector-on-the-same-node)                                                  |
  | `liveProcessCollection`, `processDiscovery`, `DD_DISCOVERY_ENABLED`                    | No exact equivalent: disabled in the reference to keep the Datadog Agent to its enabler role. Keep them if the user uses them                                                                                        |

  The reference configuration disables the Datadog Agent's Kubernetes events collection although the Collectors don't
  collect them. To collect Kubernetes events, enable the `kubernetesEvents` preset on the `cluster` collector, or keep
  `eventCollection.collectKubernetesEvents` enabled in the Datadog Agent:

  ```yaml
  spec:
    features:
      clusterChecks: { enabled: false }
      orchestratorExplorer: { enabled: false }
      kubeStateMetricsCore: { enabled: false }
      liveProcessCollection: { enabled: false }
      processDiscovery: { enabled: false }
      eventCollection: { collectKubernetesEvents: false }
      logCollection: { enabled: false }
    override:
      nodeAgent:
        containers:
          agent:
            env:
              - name: DD_ORCHESTRATOR_EXPLORER_KUBELET_CONFIG_CHECK_ENABLED
                value: "false"
              - name: DD_DISCOVERY_ENABLED
                value: "false"
              # The legacy kubernetes_state check autodiscovers kube-state-metrics and duplicates its metrics.
              - name: DD_IGNORE_AUTOCONF
                value: kubernetes_state
  ```

- **Conflicts:** if the user relies on one of these features for something the OpenTelemetry Collectors don't collect,
  keep it, and say so in the report. If the user prefers to keep a Datadog Agent feature, disable its equivalent in the
  Collectors instead, rather than collecting the telemetry twice. If `DD_IGNORE_AUTOCONF` is already set, append
  `kubernetes_state` to its space-separated list.

### A-02: Keep the Datadog Cluster Agent

- **Requirement:** don't disable the Datadog Cluster Agent: the Datadog Agent running on each node relies on it for
  cluster-level metadata, and the Datadog Operator doesn't fully support running without it
  ([datadog-operator#3521][dd-operator-3521]).

### A-03: Enable Live Debugger and Dynamic Instrumentation

- **Requirement:**

  ```yaml
  spec:
    features:
      dynamicInstrumentation: { enabled: true }
    override:
      nodeAgent:
        containers:
          agent:
            env:
              # Required by Live Debugger, as log collection is disabled.
              - name: DD_APM_DEBUGGER_LOGS_ENABLED_OVERRIDE
                value: "true"
  ```

- **Note:** Live Debugger for Go needs the system-probe module that `dynamicInstrumentation` enables, and Linux
  kernel >= 5.17 ([Live Debugger for Go][dd-live-debugger-go]).

### A-04: Disable single-step APM instrumentation, or exclude OpenTelemetry-instrumented pods

- **Requirement:** `spec.features.apm.instrumentation.enabled: false` when the OpenTelemetry Operator instruments all
  the workloads.
- **Conflicts:** when the user keeps single-step APM instrumentation for some workloads (hybrid OpenTelemetry and
  Datadog instrumentation, for example during a migration), keep it enabled, and make sure every
  OpenTelemetry-instrumented pod has the `admission.datadoghq.com/enabled: "false"` label
  ([O-05](#o-05-annotate-workloads-and-prevent-double-instrumentation)). A pod must never be instrumented by both.

### A-05: Log in JSON

- **Recommended:** `DD_LOG_FORMAT_JSON=true` in `spec.override.nodeAgent.env` and `spec.override.clusterAgent.env`, so
  that the `daemon` collector, which tails the container logs, parses the Datadog Agent's logs. See the
  `datadog-json-*` operators of `collectors.daemon.config.receivers.file_log` in [`values.yaml`][values].

### A-06: Use the same Kubernetes cluster name as the OpenTelemetry Collectors

- **Requirement:** `spec.global.clusterName` equals the OpenTelemetry Collectors' `k8s.cluster.name`
  ([C-10](#c-10-use-the-same-kubernetes-cluster-name-everywhere)). On EKS, GKE, and AKS, it can be omitted so the
  Datadog Agent detects it, unless [Fleet Automation][dd-fleet-automation] Remote Configuration is enabled, which
  requires it.
- **Recommended on non-cloud Kubernetes clusters:** use `<node name>-<cluster name>` as the Datadog Agent hostname,
  matching the `datadog.host.name` value described in
  [C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs):

  ```yaml
  spec:
    override:
      nodeAgent:
        env:
          # Kubernetes expands references to variables defined earlier in this list.
          - name: K8S_NODE_NAME
            valueFrom:
              fieldRef: { fieldPath: spec.nodeName }
          - name: K8S_CLUSTER_NAME
            value: "<CLUSTER_NAME>"
          - name: DD_HOSTNAME
            value: "$(K8S_NODE_NAME)-$(K8S_CLUSTER_NAME)"
  ```

## Known gaps

Report these as `blocked` when the user's setup depends on them:

- **App & API Protection over OTLP:** with the DDOT SDK for Go exporting over OTLP, App & API Protection data isn't
  sent to Datadog (OTEL-3411).
- **OpenTelemetry API in Go compiled with Orchestrion:** spans created with the OpenTelemetry API aren't bridged to the
  DDOT SDK for Go ([orchestrion#936][orchestrion-936]). Check whether the selected stable `dd-trace-go/v2` release
  includes that fix, and verify the bridge before removing the block.
- **Live Debugger for Go on arm64 with Linux >= 6.18:** Go probes fail to load with
  `invalid bpf_context access off=328 size=8` ([datadog-agent#57314][dd-agent-57314]). Use x86_64 nodes or an older
  arm64 kernel.

## Verify and report

Build and run each resulting container with the same command and runtime that production uses. Generate identifiable
traffic and test telemetry. Use a unique validation resource attribute, and emit a known span, metric point, and log
record for each enabled signal.

Verify all of the following:

1. **Activation:** process arguments, startup logs, or SDK diagnostics prove that one intended loader, profiler,
   runtime agent, or compiled integration is active.
2. **Effective configuration:** inspect the running container's environment and SDK diagnostics. Confirm service
   identity, whether application code or the injected SDK registers the `TracerProvider`, `MeterProvider`, and
   `LoggerProvider` and configures their exporters, enabled signals, destinations, and shared versus signal-specific
   precedence. Redact headers and credentials from retained evidence.
3. **Collector receipt:** Collector telemetry and temporary debug output show the unique test data for every enabled
   signal. A healthy receiver without the test record is insufficient.
4. **Datadog receipt:** find the same test data in Trace Explorer, Metrics Explorer, and Log Explorer as applicable.
   Check `service`, `env`, `version`, and expected semantic-convention attributes.
5. **Runtime metrics:** when enabled, find runtime-specific metric names in Datadog and record whether they traveled by
   OTLP or DogStatsD. Successful application metrics do not establish runtime metrics delivery.
6. **No duplicates:** confirm one expected trace tree per request, one copy of the marked log record, and the expected
   metric series and points. Check both Collector and Datadog Agent paths.
7. **Idempotence:** run the coding agent against the adapted repository again. It must report the setup as satisfied
   without changing dependencies, activation, global `TracerProvider`, `MeterProvider`, or `LoggerProvider`
   registration, exporter configuration, or deployment configuration.

For Kubernetes deployments, also verify:

1. **Rendered configuration:** run `helm template` with the deployment values, or
   `kubectl apply --dry-run=server`. Compare the result with [`examples/default/rendered/`][examples-rendered].
2. **Runtime resources:** confirm that every Collector, OpenTelemetry Operator, Datadog Operator, and Datadog Agent
   component present in the selected architecture is healthy. A deployment without a Datadog Agent and manual
   instrumentation without an Operator are valid. Confirm that each instrumented pod has the intended DDOT SDK and
   effective `OTEL_*` and required `DD_*` environment.

   ```sh
   kubectl get pods --namespace <collectors-namespace>
   kubectl get pods --namespace datadog
   kubectl get pod <instrumented-pod> --namespace <namespace> -o yaml | grep -E 'image:|OTEL_|DD_'
   ```

3. **Platform data in Datadog:** for components present in the architecture, confirm that [Fleet
   Automation][dd-fleet-automation] lists the Datadog Agents and Collectors under the Kubernetes cluster name and that
   **OpenTelemetry Collector** dashboard shows no refused or failed telemetry. Confirm that the services appear in APM
   with their environment and Kubernetes tags.

Record a result for every attempted application:

| Field | Required evidence |
|---|---|
| Source | Application revision, guide revision, fixture revision, and starting state |
| Versions | Runtime, operating system, architecture, SDK package or image digest, Collector, and Datadog Agent |
| Activation | Exact build and startup mechanism and evidence that it loaded |
| Configuration | Effective non-secret destination, protocol, enabled signals, resource identity, and whether application code or an injected SDK registers the `TracerProvider`, `MeterProvider`, and `LoggerProvider` and configures their exporters |
| Traces | Collector evidence, Datadog evidence, semantic attributes, and duplicate check |
| Application metrics | Collector evidence, Datadog evidence, temporality, and duplicate check |
| Application logs | Collector evidence, Datadog evidence, correlation, and duplicate check |
| Runtime metrics | Enabled state, transport, metric names, Datadog evidence, or explicit reason it was not enabled |
| Conflicts | Deliberate disables, secondary exporters, `TracerProvider`, `MeterProvider`, or `LoggerProvider` registration in application code, activation mechanisms, and preserved decisions |
| Outcome | `passed`, `failed`, `blocked`, or `untested`, with the reason |

A blocked or untested signal is not a pass. Limit the final claim to the recorded language, versions, runtime,
operating system, architecture, deployment type, and signals. See [TROUBLESHOOTING.md][troubleshooting] when telemetry
does not appear.

[install]: ./INSTALL.md
[values]: ./values.yaml
[dd-agent-yaml]: ./datadog-agent.yaml
[examples-rendered]: ./examples/default/rendered/
[troubleshooting]: ./TROUBLESHOOTING.md
[architecture]: ./assets/deployment-architecture-diagram_k8s_ddot-with-dd-agent.svg.png
[cluster-name]: https://github.com/DataDog/opentelemetry-examples/blob/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/README.md#kubernetes-cluster-name-constraints
[dd-fleet-automation]: https://docs.datadoghq.com/agent/fleet_automation/
[dd-otel-extension-resource]: https://docs.datadoghq.com/opentelemetry/integrations/datadog_extension/
[dd-live-debugger-go]: https://docs.datadoghq.com/tracing/live_debugger/?prog_lang=go
[dd-dbm]: https://docs.datadoghq.com/database_monitoring/
[dd-dsm]: https://docs.datadoghq.com/data_streams/
[dd-aap]: https://docs.datadoghq.com/security/application_security/
[dd-iast]: https://docs.datadoghq.com/security/code_security/iast/
[dd-sca]: https://docs.datadoghq.com/security/code_security/software_composition_analysis/
[orchestrion]: https://github.com/DataDog/orchestrion
[orchestrion-936]: https://github.com/DataDog/orchestrion/issues/936
[dd-operator-3521]: https://github.com/DataDog/datadog-operator/issues/3521
[dd-agent-57314]: https://github.com/DataDog/datadog-agent/issues/57314
[otel-operator-sdk-go]: https://github.com/open-telemetry/opentelemetry-operator/blob/v0.159.0/internal/instrumentation/sdk.go
[dd-correlate-metrics-traces]: https://docs.datadoghq.com/opentelemetry/correlate/metrics_and_traces/
[dd-docker-metrics-correlate]: https://docs.datadoghq.com/opentelemetry/integrations/docker_metrics/#correlate-traces-with-container-metrics
[dd-k8s-otel]: https://docs.datadoghq.com/containers/kubernetes/opentelemetry/
[dd-collector-k8s-manifest]: https://docs.datadoghq.com/opentelemetry/setup/collector_exporter/?tab=kubernetesmanifestreference#2-configure-and-deploy-the-collector
[dd-unified-service-tagging]: https://docs.datadoghq.com/getting_started/tagging/unified_service_tagging/?tab=kubernetes#configuration
[otel-semconv-k8s-attributes]: https://opentelemetry.io/docs/specs/semconv/non-normative/k8s-attributes/
[dd-hostname-mapping]: https://docs.datadoghq.com/opentelemetry/mapping/hostname/?tab=datadogexporter
[dd-collector-setup]: https://docs.datadoghq.com/opentelemetry/setup/collector_exporter/
[dd-otel-env]: https://docs.datadoghq.com/opentelemetry/config/environment_variable_support/
[dd-otel-api]: https://docs.datadoghq.com/opentelemetry/instrument/dd_sdks/api_support/
[dd-otlp-export]: https://docs.datadoghq.com/opentelemetry/instrument/dd_sdks/otlp_trace_export/
[dd-runtime-metrics]: https://docs.datadoghq.com/tracing/metrics/runtime_metrics/
[dotnet-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/dotnet-core/
[go-setup]: https://docs.datadoghq.com/tracing/trace_collection/automatic_instrumentation/dd_libraries/go/
[java-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/java/?tab=wget
[node-setup]: https://docs.datadoghq.com/tracing/trace_collection/automatic_instrumentation/dd_libraries/nodejs/
[python-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/python/
