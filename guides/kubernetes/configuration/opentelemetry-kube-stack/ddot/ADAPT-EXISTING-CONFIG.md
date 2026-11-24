# Adapt an existing setup to DDOT

Use this guide to adapt existing application instrumentation and Kubernetes telemetry infrastructure to DDOT. It
supports applications with no instrumentation, upstream OpenTelemetry, existing Datadog instrumentation, or a mixture
of these. This path adapts the existing installation without reinstalling it with the [`install` script][install].

The public Datadog tracing documentation owns language-specific SDK installation and activation. This guide owns the
migration decisions that span those SDKs: inventory the existing setup, choose one instrumentation path, move compatible
configuration into environment variables, preserve intentional features, and verify each signal. The Kubernetes rules
then adapt the OpenTelemetry Collector, OpenTelemetry Operator, and Datadog Agent. The application workflow applies
to every deployment. Apply the remaining rules only when the corresponding Kubernetes component is present.

Each Kubernetes requirement is a rule with an ID, what to check, how to fix it, and how it can conflict with an existing
setup. [`values.yaml`][values] and [`datadog-agent.yaml`][dd-agent-yaml] are the executable reference for this
repository's Kubernetes architecture. They do not replace intentional choices in another valid architecture.

Contents:

- [Instructions for coding agents](#instructions-for-coding-agents)
- [Adapt applications](#adapt-applications)
- [Target architecture](#target-architecture)
- [OpenTelemetry Collector rules](#opentelemetry-collector-rules)
- [OpenTelemetry Operator and `Instrumentation` rules](#opentelemetry-operator-and-instrumentation-rules)
- [DDOT SDK environment rules](#ddot-sdk-environment-rules)
- [Datadog Agent and Datadog Operator rules](#datadog-agent-and-datadog-operator-rules)
- [Known gaps](#known-gaps)
- [Verify and report](#verify-and-report)

## Instructions for coding agents

Run the coding agent from the repository that owns the applications and deployment configuration. Give it this prompt:

> [!TIP]
> ```text
> Adapt this repository's applications and telemetry configuration to DDOT, following
> https://raw.githubusercontent.com/DataDog/opentelemetry-examples/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/ADAPT-EXISTING-CONFIG.md
> Inspect existing instrumentation first. Prefer OpenTelemetry APIs and standard configuration.
> Preserve intentional disables and exporters, and report conflicts before changing them.
> Verify each enabled signal through the OpenTelemetry Collector and in Datadog.
> Report blocked or untested checks explicitly.
> ```

1. **Adapt each application.** Record its starting point, SDK activation, whether application code or an injected SDK
   registers the `TracerProvider`, `MeterProvider`, and `LoggerProvider` and configures their exporters, effective
   environment, destinations, and enabled signals. For the applications that already report to Datadog APM, also
   record the `service`, `env`, and `version` in effect, from the startup configuration line of the Datadog SDK or
   from Datadog APM, and the default `env` of the Datadog Agent
   ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)).
2. **Find the Kubernetes components** in the repository: OpenTelemetry Collector configurations (Helm values of the
   `opentelemetry-kube-stack` or `opentelemetry-collector` charts, `OpenTelemetryCollector` custom resources, raw
   OpenTelemetry Collector YAML), `Instrumentation` custom resources, workload manifests with
   `instrumentation.opentelemetry.io/*` annotations or `OTEL_*`/`DD_*` environment variables, `DatadogAgent` custom
   resources, and Helm values of the `datadog` or `datadog-operator` charts. Only apply the rule sections of the
   components you found in the repository or the cluster, or that the user confirmed.
   The Datadog Agent, the OpenTelemetry Collectors, and the OpenTelemetry Operator often live in other repositories,
   owned by a platform team. When you have access to the cluster, also find them there, with read-only commands:

   ```sh
   # An error that the resource type doesn't exist means that its operator isn't installed
   kubectl get datadogagents,opentelemetrycollectors,instrumentations --all-namespaces
   # Releases of the datadog, datadog-operator, opentelemetry-kube-stack, opentelemetry-collector,
   # and opentelemetry-operator charts
   helm list --all-namespaces
   # Components installed another way: look for the datadog/agent, DDOT Collector, and opentelemetry-collector images
   kubectl get daemonsets,deployments --all-namespaces \
     -o custom-columns='NS:.metadata.namespace,NAME:.metadata.name,IMAGES:.spec.template.spec.containers[*].image'
   # With Argo CD: the repository that deploys each component, which tells who owns it
   kubectl get applications.argoproj.io --all-namespaces
   ```

   Without access to the cluster, ask the user how the Datadog Agent is installed, with the Datadog Operator or the
   `datadog` Helm chart, and its version, whether an OpenTelemetry Collector or the OpenTelemetry Operator exists, and
   who owns each of them. Until the user answers, report the rules that depend on them as `blocked`. Apply the rules of
   the components that live in other repositories as `blocked` changes for their owner (step 7): don't skip them.
   Before the workloads export OTLP, check that the OTLP endpoint answers from their namespace. For a Service
   endpoint, send an empty OTLP request from a one-off pod: any HTTP status code proves that the endpoint is
   reachable, and `000` means that it isn't:

   ```sh
   kubectl run otlp-check --rm -it --restart=Never --namespace <namespace> --image=curlimages/curl -- \
     curl -s -o /dev/null -w '%{http_code}\n' -X POST -H 'Content-Type: application/json' -d '{}' \
     http://<OTLP endpoint host>:4318/v1/traces
   ```

   For the node IP endpoint of
   [S-12](#s-12-send-otlp-to-the-datadog-agent-instead-of-standalone-opentelemetry-collectors), which differs per
   node, switch one canary workload first, and check that its traces arrive in Datadog.
   For Docker Compose or another non-Kubernetes deployment, find the application commands and environment,
   OpenTelemetry Collector service and configuration, and any Datadog Agent service, then apply only the application
   workflow and public OpenTelemetry Collector setup.
   When part of the configuration lives in other repositories, for example a base image that a platform team
   maintains, or a shared Helm chart, inspect what's deployed instead: the configuration of the built image, such as
   its `ENTRYPOINT`, `CMD`, and environment, with `docker inspect` or the image registry, and a running pod, with
   `kubectl exec` as in [S-09](#s-09-configure-a-java-agent-bundled-in-the-container-image).
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
   missing value. Report a change that another repository or team owns as `blocked`, with the owner and the change
   needed, for example "base image: remove `${JAVA_TOOL_OPTIONS}` from the `java` command of the startup script
   (S-09)", instead of skipping it.

## Adapt applications

Apply this workflow to each application before changing its deployment infrastructure. Prefer OpenTelemetry APIs and
standard configuration wherever possible to keep application instrumentation portable and minimize vendor lock-in.

### 1. Inspect the existing setup

Record the runtime, architecture, package manager, lockfile, resolved SDK versions, process command, and image build.
Find every instrumentation activation mechanism. For traces, metrics, and logs, find code that creates or globally
registers a `TracerProvider`, `MeterProvider`, or `LoggerProvider`, and find code or configuration that selects an
exporter, such as OTLP. Also find every `OTEL_*` and `DD_*` setting, OpenTelemetry Collector or Datadog Agent
destination, and direct telemetry path. Include configuration in application code, container environments, Compose
files, Kubernetes manifests, ConfigMaps, Secrets, image `ENV` instructions, and startup scripts.

Classify the application and choose its migration path:

| Starting point | Action |
|---|---|
| No instrumentation | Install and activate one DDOT SDK. |
| Upstream OpenTelemetry | Keep API instrumentation. Determine whether application code or auto-instrumentation creates each `TracerProvider`, `MeterProvider`, or `LoggerProvider` and configures its exporter before replacing the upstream OpenTelemetry SDK setup. |
| Datadog instrumentation | Reuse the installed Datadog SDK when it can be converted by configuration into a DDOT SDK. See [S-13](#s-13-migrate-workloads-from-the-datadog-sdks). |
| Mixed instrumentation | Resolve components that instrument the same libraries or both register a global `TracerProvider`, `MeterProvider`, or `LoggerProvider` for the same signal before enabling another loader. |

Preserve explicit disables and intentional secondary exporters until the user approves a change. Prevent duplicate
activation, such as two Java agents, two Node.js initialization hooks, a CLR profiler with another injection mechanism,
or a Go binary compiled by two instrumentation tools.

### 2. Install or reuse one SDK

Use the public setup documentation below only to install and activate the selected SDK. Use the
[OTLP export support matrix][dd-otlp-export] to configure its transport. We recommend deploying a Datadog Agent
alongside the OpenTelemetry setup to enable Datadog-specific capabilities. The basic OTLP path works without one;
Datadog Agent-backed features require it, as described in
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
loader, profiler, runtime agent such as a Java agent, or compile-time tool activates instrumentation. For Operator
injection, take the current image reference from `instrumentation.<language>.image` in [`values.yaml`][values].

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
once to the OpenTelemetry Collector and configure an exporter for each backend there when possible. This avoids
maintaining two in-process export pipelines for the same signal. If a custom in-process processor, reader, or exporter
has no DDOT or OpenTelemetry Collector equivalent, report the migration as blocked. For Operator injection, check
whether the injected SDK registers the `TracerProvider`, `MeterProvider`, or `LoggerProvider` before retaining
application bootstrap code for that signal.

Set the service identity explicitly and use the same values for every signal:

```text
OTEL_SERVICE_NAME=<SERVICE_NAME>
OTEL_RESOURCE_ATTRIBUTES=service.namespace=<NAMESPACE>,service.version=<VERSION>,deployment.environment.name=<ENVIRONMENT>
```

On Kubernetes, follow the [OpenTelemetry Kubernetes resource-attribute conventions][otel-semconv-k8s-attributes].
Set `deployment.environment.name` in the OpenTelemetry Collector's ingestion pipeline, and only in
`OTEL_RESOURCE_ATTRIBUTES` when the OpenTelemetry Collector can't set it, or when one configuration automation sets
it in both ([C-11](#c-11-set-the-deployment-environment)).
Prefer using the OpenTelemetry Operator to inject these attributes from pod-template annotations. For manually
configured SDKs, align the environment variables with the pod annotations so application telemetry and telemetry
collected by the OpenTelemetry Collector use the same identity
([C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs)).

When migrating existing Datadog instrumentation to OpenTelemetry mode, migrate `DD_SERVICE`, `DD_ENV`, `DD_VERSION`,
and compatible `DD_TAGS` values to `OTEL_SERVICE_NAME` and `OTEL_RESOURCE_ATTRIBUTES`. On Kubernetes, also align the
Unified Service Tagging labels ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)). Convert Datadog tag syntax to
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

For an SDK configured manually, use one shared endpoint when traces, metrics, and logs go to the same local
OpenTelemetry Collector:

```text
OTEL_EXPORTER_OTLP_ENDPOINT=http://<COLLECTOR_HOST>:4318
```

**Advanced configuration:** use a signal-specific endpoint only when that signal has a different destination.
Signal-specific endpoints override the shared endpoint. For OTLP/HTTP, include the signal path, such as `/v1/traces`,
`/v1/metrics`, or `/v1/logs`. For OTLP/gRPC, use the host and port without an HTTP signal path.

The destination pipeline depends on the deployment. This repository's reference architecture exports from the DDOT
Collector to Datadog with the Datadog exporter. Preserve intentional secondary exporters and custom routing, and verify
that no path sends a second copy of the same signal. Preserve a compatible existing export path instead of replacing it
only to match this repository. For a new or different OpenTelemetry Collector deployment, follow the
[current Datadog OpenTelemetry Collector setup][dd-collector-setup].

We recommend deploying a Datadog Agent alongside the OpenTelemetry setup to enable Datadog-specific capabilities.
The basic OTLP path from the SDK to an OpenTelemetry Collector works without a Datadog Agent.
Features such as Remote Configuration, Live Debugger, Dynamic Instrumentation, and DogStatsD use the Datadog Agent.
Set `DD_AGENT_HOST` to the reachable Datadog Agent. Allow SDK access to TCP `8126` for Datadog Agent-backed features
and, when runtime metrics use DogStatsD, UDP `8125`. Preserve these paths when switching trace export to OTLP; see
[Datadog Agent network configuration][dd-agent-network]. The Datadog SDK
[runtime-metrics guide][dd-runtime-metrics] owns the language-specific defaults and metric names. If the application
sends OTLP directly to a Datadog Agent, enable its OTLP receiver and use Datadog Agent 7.48.0 or later for logs
([S-12](#s-12-send-otlp-to-the-datadog-agent-instead-of-standalone-opentelemetry-collectors)). Avoid
sending the same signal through both OpenTelemetry Collector and Datadog Agent paths.

Switch trace export from the native Datadog Agent path to OTLP only after OTLP trace delivery succeeds. Do not disable
the tracer; follow the [OTLP export documentation][dd-otlp-export] for the exporter switch. Review direct log
submission, file tailing, DogStatsD, and runtime metrics independently because they do not all follow the trace
exporter. Send each log line once: see [S-10](#s-10-send-each-log-line-to-datadog-once). For a Java agent bundled in
the image, see [S-09](#s-09-configure-a-java-agent-bundled-in-the-container-image).

## Target architecture

![Deployment architecture: the OpenTelemetry Kube Stack deploys the DDOT SDKs, OpenTelemetry Collectors, and host profiler, next to a Datadog Agent acting as an enabler of the DDOT SDKs][architecture]

After the changes in this guide:

- **OpenTelemetry Collectors** collect the Kubernetes and application telemetry and export it to Datadog. A
  `daemon` OpenTelemetry Collector (DaemonSet) receives the applications' OTLP traces, metrics, and logs, and collects node-scope
  telemetry, including the telemetry of the middleware running on its node. A `cluster` OpenTelemetry Collector (single-replica Deployment) collects cluster-scope telemetry.
- **The OpenTelemetry Operator** injects the DDOT SDKs into annotated workloads through an `Instrumentation` custom
  resource, and points them at the `daemon` OpenTelemetry Collector.
- **The DDOT SDKs** export OTLP to the `daemon` OpenTelemetry Collector.
- **The Datadog Agent** only acts as an **enabler of the DDOT SDKs**: it enables their Datadog-specific features, such
  as Live Debugger, Dynamic Instrumentation, Remote Configuration of head sampling, or Feature Flag Management. It
  doesn't collect the Kubernetes telemetry or instrument the applications, so its overlapping features are disabled.

## OpenTelemetry Collector rules

These rules apply to the OpenTelemetry Collector configuration, whatever the way it's deployed. The paths refer to the
`opentelemetry-kube-stack` chart, where `defaultCRConfig` is merged into every OpenTelemetry Collector and `collectors.<name>`
configures one OpenTelemetry Collector.

### C-01: Export to Datadog with the Datadog exporter

- **Requirement:** a `datadog` exporter, with the API key and site read from a Kubernetes Secret.
- **Fix:**

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.env snippet-path=.env mode=subset sha=4265b637f893 -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.exporters["datadog/exporter"] snippet-path=.config.exporters["datadog/exporter"] mode=subset sha=34f7be239670 -->
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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.collectors.daemon.config.connectors snippet-path=.connectors mode=subset sha=4d1d2d7b4ca1 -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.collectors.daemon.config.service.pipelines mode=adapted sha=49a13dd416a5 -->
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
- **Conflicts:** when the SDKs sample the traces, the connector only sees the sampled ones. It scales the APM trace
  metrics back up with the sampling probability in the W3C `tracestate` from Datadog Agent 7.82.0, for the DDOT
  Collector and the bundled DDOT Collector. The Datadog SDKs that export traces over OTLP don't get the sampling rates
  of the Datadog Agent, which come in its responses to the traces sent to port 8126, and can keep all the traces
  ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)): add the sampling processors here when the ingested volume
  must stay limited.

### C-03: Enrich the telemetry with the resource attributes Datadog needs

- **Requirement:** the traces, metrics, and logs of a workload carry the same values of the following resource
  attributes: Datadog correlates the telemetry on them, and can't link the signals of a workload when they differ.
  `service.namespace` and `deployment.environment.name` are strongly recommended rather than required.

  | Resource attribute                                                                                                                                                                                                                   | Datadog use                                                                                           | Set by, in the reference configuration                                                                                                                                   |
  |--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
  | `service.name`                                                                                                                                                                                                                       | `service` tag, needed for any correlation                                                             | OpenTelemetry Operator and `k8s_attributes`, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]: annotations, labels, workload names      |
  | `service.namespace`                                                                                                                                                                                                                  | Groups the services of an application. Strongly recommended, not required                             | OpenTelemetry Operator and `k8s_attributes`, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]: annotations, Kubernetes namespace        |
  | `deployment.environment.name`                                                                                                                                                                                                        | `env` tag                                                                                             | OpenTelemetry Operator and `resourceAttributes` of the kube-stack chart ([C-11](#c-11-set-the-deployment-environment)): annotations, OpenTelemetry Collector pipeline             |
  | `service.version`                                                                                                                                                                                                                    | `version` tag                                                                                         | OpenTelemetry Operator and `k8s_attributes`, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]: annotations, labels, container image tag |
  | `container.id`                                                                                                                                                                                                                       | Links spans to the container metrics: without it, APM doesn't show the container metrics of a service | The SDK's container resource detector: not extracted by `k8s_attributes` in the reference                                                                                |
  | `host.name`                                                                                                                                                                                                                          | Links to the host metrics                                                                             | `resource/hostname`, from `k8s.node.name`                                                                                                                                |
  | `host.id`, `cloud.provider`, `cloud.account.id`                                                                                                                                                                                      | Datadog hostname of cloud hosts, cloud context                                                        | `resource_detection` with the cloud provider's detector                                                                                                                  |
  | `datadog.host.name`                                                                                                                                                                                                                  | Datadog hostname, when `host.id` can't be reliably discovered                                         | Not set in the reference as `k8s.cluster.name` is set when it can't be discovered                                                                                        |
  | `k8s.pod.name`                                                                                                                                                                                                                       | Links the pod and container views                                                                     | OpenTelemetry Operator and OpenTelemetry Operator, `k8s_attributes`                                                                                                      |
  | `k8s.container.name`                                                                                                                                                                                                                 | Container name displayed in Datadog                                                                   | OpenTelemetry Operator (`OTEL_RESOURCE_ATTRIBUTES`), `k8s_attributes` for the other workloads                                                                            |
  | `k8s.cluster.name`                                                                                                                                                                                                                   | Kubernetes Explorer, Kubernetes dashboards                                                            | `resource_detection` (EKS, GKE, AKS) or `resourceAttributes` ([C-10](#c-10-use-the-same-kubernetes-cluster-name-everywhere))                                             |
  | `k8s.namespace.name`, `k8s.node.name`, `k8s.pod.uid`, `k8s.deployment.name`, `k8s.replicaset.name`, `k8s.daemonset.name`, `k8s.statefulset.name`, `k8s.job.name`, `k8s.cronjob.name`, `container.image.name`, `container.image.tags` | Kubernetes context of the telemetry                                                                   | `k8s_attributes`                                                                                                                                                         |

- **Fix:** enrich the telemetry in the OpenTelemetry Collector with the `k8s_attributes` processor, in every pipeline that receives
  telemetry from the workloads, rather than setting these attributes by hand in each application:
  - with the OpenTelemetry Kube Stack Helm chart, `presets.kubernetesAttributes.enabled: true`. It extracts the
    attributes above, except `container.id` and `k8s.cluster.name`, and sets `otel_annotations: true`;
  - otherwise, the `k8s_attributes` processor with its RBAC (`get`, `list`, and `watch` on pods, namespaces, and
    replica sets, and on nodes when you extract node metadata, such as `k8s.node.uid` or node labels), and the
    `metadata` to extract:

    <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.processors.k8s_attributes.extract.metadata snippet-path=.processors.k8s_attributes.extract.metadata mode=subset sha=4221395d5d6f -->
    <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.processors.k8s_attributes mode=adapted sha=179ff668cf20 -->
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
            - container.image.tags
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
    metrics and logs that the OpenTelemetry Collectors collect.
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
  - **`deployment.environment.name`:** set it in the OpenTelemetry Collector's ingestion pipeline, which is strongly
    recommended. Only set it in the SDKs when the OpenTelemetry Collector can't set it, or when one configuration
    automation, such as the OpenTelemetry Kube Stack Helm chart, sets it in both
    ([C-11](#c-11-set-the-deployment-environment)).
  - **`k8s.node.name`:** set it with the `k8s_attributes` processor, from the pod the telemetry comes from.
  - **`host.id`, `cloud.provider`, and `cloud.account.id`:** set them whenever the OpenTelemetry Collector can discover them,
    typically with the `resource_detection` processor and the cloud provider's detector (`eks`, `aks`, `gke`...), so
    that Datadog attaches the telemetry to the right host
    ([hostname mapping][dd-hostname-mapping]). Check that the detector enables these attributes in its
    `resource_attributes`, and only set `host.id` on the telemetry of the OpenTelemetry Collector's own node, that is, in the
    `daemon` OpenTelemetry Collector behind a node-local Service ([C-13](#c-13-keep-the-daemon-opentelemetry-collectors-service-node-local)): not
    on the cluster-scope telemetry of the `cluster` OpenTelemetry Collector, which describes other nodes.
  - **On EKS:** enable these resource attributes of the `eks` detector, because several of them are disabled by default.
    The OpenTelemetry Kube Stack Helm chart's preset (`presets.resourceDetection.eks.enabled: true`) only enables the
    detector, so the [`install`][install] script also sets these attributes with `--set` flags on `defaultCRConfig` when
    the cluster is EKS:

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
    `datadog.host.name` to `<node name>-<cluster name>` in the `daemon` OpenTelemetry Collector, the same value as the Datadog
    Agent's hostname ([A-06](#a-06-use-the-same-kubernetes-cluster-name-as-the-opentelemetry-collectors)), so that the
    Datadog Agent and the OpenTelemetry Collectors report the same host:

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
  OpenTelemetry Kube Stack Helm chart's preset does. The OpenTelemetry Operator injects it in `OTEL_RESOURCE_ATTRIBUTES`
  only for the workloads it instruments: `k8s_attributes` sets it for the other ones, such as applications instrumented
  without the Operator. For those applications, also set it in the container's `OTEL_RESOURCE_ATTRIBUTES`
  ([S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)). In pods with several containers,
  the processor can only tell which container sent the telemetry from its `container.id` resource attribute, and doesn't
  set `k8s.container.name` without it.
- **`container.id`:** check that the application traces carry it. The reference configuration has `container.id` in the
  `k8s_attributes` processor's `extract.metadata`: the processor then looks up the container ID from the pod's status,
  using the `k8s.container.name` that the OpenTelemetry Operator injects in `OTEL_RESOURCE_ATTRIBUTES`.
- **`container.image.tags`:** extract it with the `k8s_attributes` processor. Only extract the deprecated
  `container.image.tag` for the Collectors and the consumers of the telemetry that don't support
  `container.image.tags` yet:
  - **History:** [semantic conventions v1.22.0][otel-semconv-v1-22-0] (October 2023) replaced the
    `container.image.tag` string attribute with the `container.image.tags` string array. The `k8s_attributes` processor
    added `container.image.tags` in Collector v0.146.0, with two feature gates that [switch the processor to the stable
    Kubernetes conventions][otel-k8sattributes-semconv]: `processor.k8sattributes.EmitV1K8sConventions` emits the
    stable attribute names, and `processor.k8sattributes.DontEmitV0K8sConventions` stops emitting the legacy names.
    The same gates also rename the label and annotation attributes, for example `k8s.pod.labels.<key>` to
    `k8s.pod.label.<key>`.
  - **Since Collector v0.161.0:** both feature gates are enabled by default. `k8s_attributes` emits
    `container.image.tags`, and ignores `container.image.tag` in `extract.metadata` with a startup warning:
    `container.image.tag is configured but will NOT be emitted`. The kube-stack chart's `kubernetesAttributes` preset
    still extracts `container.image.tag` ([opentelemetry-helm-charts#2538][otel-helm-charts-2538]), so the reference
    configuration replaces the preset's list in `defaultCRConfig.config.processors.k8s_attributes.extract.metadata` and
    `collectors.host-profiler.config.processors.k8s_attributes.extract.metadata` in [`values.yaml`][values].
  - **Datadog:** since Collector v0.161.0, the Datadog exporter and connector map `container.image.tags` to the
    `image_tag` tag, using the first element of the array. Earlier versions only map `container.image.tag`. The DDOT
    Collector maps `container.image.tags` since Datadog Agent v7.85.0.
  - **When to extract `container.image.tag`:**
    - **Collector older than v0.146.0:** `container.image.tag` is the only image tag attribute that `k8s_attributes`
      supports.
    - **Collector v0.146.0 to v0.160.x:** keep `container.image.tag`. `k8s_attributes` only emits
      `container.image.tags` when `processor.k8sattributes.EmitV1K8sConventions` is enabled, and the Datadog exporter
      of these versions doesn't map it.
    - **Collector v0.161.0 or later, with consumers that still read `container.image.tag`,** such as dashboards,
      monitors, or other observability backends: emit both attributes during the migration. List both in
      `extract.metadata`, and disable the legacy gate with
      `--feature-gates=-processor.k8sattributes.DontEmitV0K8sConventions`.
- **Conflicts:** `k8s_attributes` doesn't overwrite attributes the telemetry already has. Remove the hand-set values
  that differ from the Kubernetes metadata, such as a `host.name` set to the container's hostname by the SDK's host
  detector, rather than keeping two sources of truth.
- **Docs:** [Correlate OpenTelemetry traces and metrics][dd-correlate-metrics-traces],
  [correlate traces with container metrics][dd-docker-metrics-correlate],
  [Monitor Kubernetes with OpenTelemetry][dd-k8s-otel], the [OpenTelemetry Collector's Kubernetes manifest
  reference][dd-collector-k8s-manifest], and [Unified Service Tagging][dd-unified-service-tagging].

FIXME verify that the Datadog OTel Contrib exporter and extension can discover `datadog.host.name` passed as an
environment variable, typically through `OTEL_RESOURCE_ATTRIBUTES=datadog.host.name=<node name>-<cluster name>`.

<a id="c-04-dont-stamp-the-collectors-own-identity-on-incoming-telemetry"></a>

### C-04: Don't stamp the OpenTelemetry Collector's own identity on incoming telemetry

- **Requirement:** no `env` detector of the `resource_detection` processor on OpenTelemetry Collectors that receive application,
  middleware, or host telemetry, and `override: false` on the `resource_detection` processors.
- **Why:** the `env` detector adds the OpenTelemetry Collector pod's own `k8s.pod.name`, `k8s.namespace.name`, `node.name`, and
  `host.name` to the telemetry it receives, overwriting the application's identity.
- **Fix:** with the OpenTelemetry Kube Stack Helm chart,
  `collectors.daemon.presets.resourceDetection.env.enabled: false` (enabled by the chart's daemon defaults), and:

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.processors["resource_detection/env"] snippet-path=.processors["resource_detection/env"] mode=subset sha=17ca21355cc7 -->
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

<a id="c-07-report-the-opentelemetry-collectors-to-fleet-automation-with-the-datadog-extension"></a>

### C-07: Report the OpenTelemetry Collectors to Fleet Automation with the Datadog extension

- **Requirement:** the `datadog` extension, with `deployment_type: gateway` on Deployments and
  `deployment_type: daemonset` on DaemonSets, listed in `service.extensions`.
- **Why:** it lists the OpenTelemetry Collectors in Datadog [Fleet Automation][dd-fleet-automation], with their version and
  configuration.
- **Fix:**

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.extensions.datadog.api snippet-path=.extensions.datadog.api mode=exact sha=b70b5c98ceb1 -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.collectors.daemon.config.extensions.datadog.deployment_type snippet-path=.extensions.datadog.deployment_type mode=exact sha=470b1905b32d -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.service.extensions snippet-path=.service.extensions mode=subset sha=4f8fc8145918 -->
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

<a id="c-08-export-the-collectors-internal-metrics-through-its-own-pipeline"></a>

### C-08: Export the OpenTelemetry Collector's internal metrics through its own pipeline

- **Requirement:** the OpenTelemetry Collector exports its internal metrics over OTLP to its own OTLP receiver, so that they get the
  same resource attributes and tags as the rest of the telemetry.
- **Why:** the internal metrics then go through the OpenTelemetry Collector's processors, including the `resource_detection`
  processor of the Kubernetes platform (EKS, AKS, GKE), and get enriched with resource attributes such as `host.id` and
  `k8s.cluster.name`. Exported directly to Datadog, they would miss these attributes, and the OpenTelemetry Collector's health
  metrics couldn't be correlated with its Kubernetes cluster and host.
- **Fix:**

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.service.telemetry snippet-path=.service.telemetry mode=subset sha=c765f168a501 -->
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

- **Conflicts:** don't do the same for the OpenTelemetry Collector's own logs when the `daemon` OpenTelemetry Collector's `filelog` receiver already
  tails the container logs: it would duplicate them.

### C-09: Receive OTLP over both gRPC and HTTP

- **Requirement:** the `otlp` receiver listens on `0.0.0.0:4317` (gRPC) and `0.0.0.0:4318` (HTTP).

### C-10: Use the same Kubernetes cluster name everywhere

- **Requirement:** the `k8s.cluster.name` resource attribute of the OpenTelemetry Collectors equals the Datadog Agent's
  `clusterName` ([A-06](#a-06-use-the-same-kubernetes-cluster-name-as-the-opentelemetry-collectors)).
- **Fix:** on EKS, GKE, and AKS, enable the cloud provider's resource detection preset, which discovers it.
- **Fix:** on other platforms, set `k8s.cluster.name` both on the telemetry and on the OpenTelemetry Collector's own
  telemetry. The OpenTelemetry Collector's Datadog extension
  ([C-07](#c-07-report-the-opentelemetry-collectors-to-fleet-automation-with-the-datadog-extension)) reports the
  OpenTelemetry Collector under the cluster name of its own telemetry in Fleet Automation. With the OpenTelemetry Kube
  Stack Helm chart, `resourceAttributes.k8s.cluster.name` in the values does both:

  ```yaml
  resourceAttributes:
    k8s.cluster.name: <CLUSTER_NAME>
  ```

  Otherwise, add a `resource` processor that upserts `k8s.cluster.name` to the pipelines, and set `k8s.cluster.name`
  under `service.telemetry.resource`, as the [Datadog extension documentation][dd-otel-extension-resource] describes.
- **Constraints:** dot-separated tokens of lowercase letters, digits, and `-`, starting with a letter and ending with a
  letter or a digit, at most 80 characters. See [Kubernetes cluster name constraints][cluster-name].

### C-11: Set the deployment environment

- **Recommendation:** strongly recommended, set `deployment.environment.name` on all the telemetry. Datadog maps it to
  the `env` tag of [Unified Service Tagging][dd-unified-service-tagging].
- **Fix: set it in the OpenTelemetry Collector's ingestion pipeline.** This is strongly recommended:
  - the setup is simple, and the application deployment manifests stay the same across environments;
  - it enriches all the telemetry that goes through the OpenTelemetry Collector: the telemetry of the SDKs, and the
    telemetry that the OpenTelemetry Collector collects itself, such as pod logs and metrics, Kubernetes events, and
    its own internal telemetry ([C-08](#c-08-export-the-collectors-internal-metrics-through-its-own-pipeline)).

  With the OpenTelemetry Kube Stack Helm chart, set `resourceAttributes` in the values:

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.resourceAttributes snippet-path=.resourceAttributes mode=exact sha=fe16d989fd8b -->
  ```yaml
  resourceAttributes:
    deployment.environment.name: <ENVIRONMENT>
  ```

  The chart adds a `resource/global` processor with `action: upsert` to every pipeline of the OpenTelemetry
  Collectors. The installer's [`values.yaml`][values] sets it to `production`, and the `install` script to its
  `DEPLOYMENT_ENVIRONMENT_NAME` variable, `production` by default. Before reusing these values, set the environment of
  each cluster, and remove the entry when workloads carry their own `deployment.environment.name`: with `upsert`, the
  value overwrites theirs, for example `production` on the workloads of a staging cluster
  ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)). Without the chart,
  add a `resource` processor to every pipeline that receives telemetry from the workloads, including the pod logs and
  metrics pipelines. A `transform` processor with
  `set(resource.attributes["deployment.environment.name"], "<ENVIRONMENT>")` also works.

  ```yaml
  processors:
    resource/environment:
      attributes:
        - key: deployment.environment.name
          value: <ENVIRONMENT>
          action: upsert   # or insert, to keep a value set by the workload
  ```

- **Fix: set it in the SDKs only when you can't set it in the OpenTelemetry Collector,** for example when one
  OpenTelemetry Collector receives the telemetry of several environments, or when you can't change its configuration.
  Without the OpenTelemetry Operator, set the `deployment.environment.name` entry of `OTEL_RESOURCE_ATTRIBUTES` and
  the `resource.opentelemetry.io/deployment.environment.name` pod annotation, with the same value
  ([S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)). The annotation lets the
  `k8s_attributes` processor set the attribute on the telemetry that the OpenTelemetry Collector collects about the
  pod.

  **Exception:** set it in both the OpenTelemetry Collector and the SDKs when one configuration automation manages both,
  such as the OpenTelemetry Kube Stack Helm chart. Its `resourceAttributes` sets the attributes in the OpenTelemetry
  Collectors and in the `Instrumentation` resource, which the OpenTelemetry Operator injects in the workloads it
  instruments, with the same values. The automation keeps the values consistent, and the telemetry gets the attributes
  from the SDK, before it reaches any OpenTelemetry Collector pipeline. This applies to `deployment.environment.name`,
  to `k8s.cluster.name` ([C-10](#c-10-use-the-same-kubernetes-cluster-name-everywhere)), and to any custom resource
  attribute that has the same value for the whole Kubernetes cluster.

  **Exception:** set it in the SDKs when the workload already has a Datadog environment, the
  `tags.datadoghq.com/env` label or `DD_ENV`: set `deployment.environment.name` in `OTEL_RESOURCE_ATTRIBUTES` and in
  the `resource.opentelemetry.io/deployment.environment.name` annotation, with the same value
  ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)). The OpenTelemetry
  Collector then only sets a default for the other workloads, with `action: insert`, and never overwrites the
  workload's value.
- **Conflicts:** when the workloads rely on the default `env` of the Datadog Agent, its `apm_config.env` or `env`
  settings, or an `env:` entry of its `tags`, keep the same value after the switch
  ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)):
  - **With a standalone OpenTelemetry Collector,** at level 1 of
    [C-14](#c-14-choose-where-the-opentelemetry-collector-runs-on-kubernetes), the traces don't go through the
    Datadog Agent and don't get its default: without `deployment.environment.name`, they get `env:none`. Set the same
    value in the OpenTelemetry Collector, as this rule describes.
  - **With the bundled DDOT Collector or OTLP ingest,** at levels 2 and 3, the Datadog Agent's trace processing
    still applies its default `env` to the traces that arrive without one. Setting the same value in the bundled DDOT
    Collector is optional.
- **Conflicts:** when both the SDKs and the OpenTelemetry Collector set it, use the same value. With `action: upsert`,
  which the OpenTelemetry Kube Stack Helm chart uses, the OpenTelemetry Collector's value overwrites a different value
  set in the SDK. To use the OpenTelemetry Collector's value as a default that workloads can override, remove
  `deployment.environment.name` from `resourceAttributes` and use a `resource` processor with `action: insert`. A
  `transform` processor gets the same result with `where resource.attributes["deployment.environment.name"] == nil`.
  Place that processor after `k8s_attributes` in each pipeline, so that a value from the
  `resource.opentelemetry.io/deployment.environment.name` annotation is already set when the default applies.
- **Verify:** in Datadog, check the `env` tag of the service in APM and of the pod's logs and container metrics. Every
  signal of the workload must have the same value.

<a id="c-12-collect-middleware-telemetry-from-the-daemon-opentelemetry-collector-on-the-same-node"></a>

### C-12: Collect middleware telemetry from the `daemon` OpenTelemetry Collector on the same node

- **Requirement:** collect the telemetry of the middleware running in the Kubernetes cluster, such as database
  servers, message brokers, or caches, with the `daemon` OpenTelemetry Collector running on the same Kubernetes node as the observed
  pod. Not with the `cluster` OpenTelemetry Collector, a standalone OpenTelemetry Collector Deployment, or a `daemon` OpenTelemetry Collector on another node.
- **Why:** the OpenTelemetry Collector enriches the telemetry it collects with the resource attributes of the observed pod, such as
  its `k8s.node.name`. Collected from another node, or through a Kubernetes Service that load balances across pods,
  the telemetry gets the wrong resource attributes, for example the OpenTelemetry Collector's node or another replica's
  `k8s.pod.name`, and is attached to the wrong host in Datadog.
- **Fix:** in the `daemon` OpenTelemetry Collector, discover the middleware pods of its own node with the `k8s_observer` extension,
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
- **Conflicts:** move existing middleware receivers from the `cluster` OpenTelemetry Collector or a standalone OpenTelemetry Collector Deployment
  to the `daemon` OpenTelemetry Collector. Keep in the `cluster` OpenTelemetry Collector the cluster-scope telemetry (kube-state-metrics,
  Kubernetes objects) and the middleware running outside the Kubernetes cluster, such as managed cloud databases,
  which has no Kubernetes node: don't set `host.name` on its telemetry from the OpenTelemetry Collector's node.

<a id="c-13-keep-the-daemon-opentelemetry-collectors-service-node-local"></a>

### C-13: Keep the `daemon` OpenTelemetry Collector's Service node-local

- **Requirement:** the `daemon` OpenTelemetry Collector's Service, the one the SDKs export to
  ([O-04](#o-04-point-the-sdks-at-the-daemon-opentelemetry-collector)), has `internalTrafficPolicy: Local`, so that each pod's
  telemetry always reaches the `daemon` OpenTelemetry Collector running on its own Kubernetes node.
- **Why:** with `internalTrafficPolicy: Cluster`, the Service load balances each connection across the `daemon`
  OpenTelemetry Collectors of all the nodes:
  - the points of a time series can reach different OpenTelemetry Collector instances, which breaks the stateful `cumulative_to_delta`
    conversion ([S-04](#s-04-export-delta-metrics));
  - the telemetry crosses nodes, and availability zones, for nothing;
  - the load of a `daemon` OpenTelemetry Collector no longer depends on the workloads of its node, so its resource allocation no
    longer matches it.
- **Detect:**

  ```sh
  kubectl get service <helm-release-name>-daemon-collector --namespace <namespace> \
    -o jsonpath='{.spec.internalTrafficPolicy}'   # expected: Local
  ```

- **Fix:** the OpenTelemetry Operator already sets `internalTrafficPolicy: Local` on the Services of the OpenTelemetry Collectors in
  `daemonset` mode, and so does the `opentelemetry-collector` Helm chart in `daemonset` mode, unless
  `service.internalTrafficPolicy` is set: remove any explicit `internalTrafficPolicy: Cluster` override. For an OpenTelemetry Collector
  DaemonSet deployed with plain manifests, set it on its Service:

  ```yaml
  apiVersion: v1
  kind: Service
  spec:
    internalTrafficPolicy: Local
  ```

- **Conflicts:** with `Local`, there's no fallback to another node: the telemetry of a pod running on a node without a
  ready `daemon` OpenTelemetry Collector is dropped. Make the `daemon` OpenTelemetry Collector run on every node running instrumented workloads,
  with the `tolerations` and `nodeSelector` matching the nodes' taints and labels. Don't apply this rule to the
  `cluster` OpenTelemetry Collector, a Deployment.

### C-14: Choose where the OpenTelemetry Collector runs on Kubernetes

- **Recommendation:** receive the telemetry of the workloads in an OpenTelemetry Collector that runs as a DaemonSet,
  on the node of each workload ([C-13](#c-13-keep-the-daemon-opentelemetry-collectors-service-node-local)). In
  decreasing order of preference:
  1. **The DDOT Collector, deployed as a DaemonSet with the OpenTelemetry community tools,** as this guide's reference
     configuration does. Deploy it preferably with the OpenTelemetry Kube Stack Helm chart, otherwise with an
     `OpenTelemetryCollector` resource of the OpenTelemetry Operator, otherwise with the
     [OpenTelemetry Collector Helm chart][otel-collector-chart], otherwise with your own deployment mechanism. When
     the DDOT Collector can't be used, the upstream OpenTelemetry Collector Contrib, with the same configuration, is
     the solution. With the OpenTelemetry Kube Stack Helm chart, this level deploys the OpenTelemetry Operator, with
     its custom resource definitions and its admission webhook, which needs [cert-manager][cert-manager] when it isn't
     installed already, and the DDOT Collectors from the chart's values. It also needs a `datadog-secret` Secret with
     the Datadog API key in their namespace ([C-01](#c-01-export-to-datadog-with-the-datadog-exporter)). Tell the user
     about this footprint when they choose between the levels.
  2. **The DDOT Collector bundled in the Datadog Agent DaemonSet,** when an OpenTelemetry Collector DaemonSet deployed
     with the community tools isn't an option. It has the same support of the OpenTelemetry Collector components as
     the standalone DDOT Collector. Add the `k8s_attributes` processor to its ingestion pipelines, and if you can't,
     rely on the `infraattributes` processor, with its limits
     ([S-12](#s-12-send-otlp-to-the-datadog-agent-instead-of-standalone-opentelemetry-collectors)).
  3. **OTLP ingest in the Datadog Agent DaemonSet,** when the bundled DDOT Collector isn't an option. Its limits are
     at least those of the bundled DDOT Collector without `k8s_attributes`
     ([S-12](#s-12-send-otlp-to-the-datadog-agent-instead-of-standalone-opentelemetry-collectors)).
- **Why:**
  - the OpenTelemetry community tools keep the deployment of the OpenTelemetry Collector free from vendor lock-in,
    and interchangeable between the DDOT and the upstream OpenTelemetry Collector;
  - the `k8s_attributes` processor with `extract.otel_annotations: true` is the only component that applies the
    [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes] to all the telemetry of a workload, including
    the pod logs and metrics that the OpenTelemetry Collector collects itself
    ([C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs)). Levels 1 and 2 can use it, but
    level 3 can't.
- **Conflicts:**
  - When the user already runs one of these setups, keep it, and report the level it corresponds to and its limits,
    instead of migrating it without the user's approval.
  - When the user only runs a Datadog Agent, without an OpenTelemetry Collector, ask the user before adopting level 1.
    With level 1 and [A-01](#a-01-disable-the-features-that-duplicate-the-opentelemetry-collectors), the OpenTelemetry
    Collectors collect the Kubernetes metrics and the pod logs instead of the Datadog Agent, with different tags, which
    can break existing infrastructure monitors and dashboards
    ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)). Levels 2 and 3 keep the Datadog Agent's infrastructure
    collection unchanged.
  - **Level 1 for APM only:** when the user adopts level 1 to migrate the SDKs, but wants to keep the Datadog Agent as
    the source of the pod logs and of the Kubernetes and host metrics, at least during the migration, turn off the
    collection of the OpenTelemetry Collectors instead of applying
    [A-01](#a-01-disable-the-features-that-duplicate-the-opentelemetry-collectors). Otherwise, both collect the same
    logs and metrics, and every log line and every one of these metrics reaches Datadog twice. In the values of the
    OpenTelemetry Kube Stack Helm chart:

    | Value                                              | Set to  | What it would duplicate                                                                  |
    |----------------------------------------------------|---------|------------------------------------------------------------------------------------------|
    | `collectors.daemon.presets.logsCollection.enabled` | `false` | The container logs, which the Datadog Agent tails                                        |
    | `collectors.daemon.presets.kubeletMetrics.enabled` | `false` | The kubelet, pod, and container metrics                                                  |
    | `collectors.daemon.presets.hostMetrics.enabled`    | `false` | The host metrics                                                                         |
    | `collectors.cluster.enabled`                       | `false` | The Kubernetes cluster metrics and objects of the `cluster` OpenTelemetry Collector     |
    | `kubeStateMetrics.enabled`                         | `false` | The kube-state-metrics deployment that the `cluster` OpenTelemetry Collector scrapes     |

    The `daemon` OpenTelemetry Collector then only receives the OTLP telemetry of the SDKs. Move the logs or the
    infrastructure metrics later, as a separate step, by turning these values on and applying A-01 together. The
    OpenTelemetry Collectors then also need a default `deployment.environment.name` for this telemetry, set with
    `action: insert` ([C-11](#c-11-set-the-deployment-environment)).

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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.dotnet.env snippet-path=.dotnet.env mode=exact sha=d713625ee722 -->
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

- **Requirement:** for Operator-managed Python injection, use the DDOT SDK for Python image from
  [`values.yaml`][values], version `4.16.1-preview` or later. These images fit the OpenTelemetry Operator's default
  200 Mi auto-instrumentation volume. Upgrade older images before using that default volume.
- **Alpine/musl workloads:** add the pod-template annotation
  `instrumentation.opentelemetry.io/otel-python-platform: "musl"` (glibc is the default). See
  [Auto-instrumenting a Python musl based container](https://opentelemetry.io/docs/platforms/kubernetes/operator/automatic/#annotations-python-musl).

<a id="o-04-point-the-sdks-at-the-daemon-opentelemetry-collector"></a>

### O-04: Point the SDKs at the `daemon` OpenTelemetry Collector

- **Requirement:** `spec.exporter.endpoint` is the `daemon` OpenTelemetry Collector Service's OTLP HTTP endpoint,
  `http://<helm-release-name>-daemon-collector.<namespace>.svc.cluster.local:4318`. This Service must be node-local
  ([C-13](#c-13-keep-the-daemon-opentelemetry-collectors-service-node-local)).
- **Requirement:** `spec.sampler.type: parentbased_always_on` and `spec.propagators: [tracecontext, baggage]`. Sample
  in the OpenTelemetry Collector instead ([C-02](#c-02-compute-apm-trace-metrics-with-the-datadog-connector-before-any-sampling)).
  During a migration from the Datadog SDKs, check that the workloads that still run Datadog SDKs propagate
  `tracecontext` ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)).
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

- **Why:** the label stops the Datadog Cluster Agent's Admission Controller from also instrumenting the pod. It must be
  a label: the Datadog Cluster Agent's Admission Controller only reads pod labels, so a `podSelector` keyed on the
  OpenTelemetry annotations never matches.

### O-06: Instrument Go applications at compile time

- **Recommendation:** instrument Go applications at build time with [Orchestrion][orchestrion] for the DDOT SDK for
  Go. Compile-time instrumentation is recommended and optional: see the alternative below.
- **Requirement:** when the OpenTelemetry Operator configures Go applications, use the
  `instrumentation.opentelemetry.io/inject-sdk` annotation. It injects configuration without an SDK, because the
  application binary already contains the DDOT SDK for Go. It injects the OpenTelemetry configuration, such as the
  exporter endpoint, resource attributes, sampler, and propagators. It also injects every variable of the
  `Instrumentation`'s `spec.env`, including the `DD_*` variables of the Datadog-specific features of the DDOT SDKs
  ([S-01](#s-01-enable-the-opentelemetry-mode-of-the-ddot-sdks),
  [S-06](#s-06-enable-optional-datadog-features-per-workload)). It doesn't inject the language sections of the
  `Instrumentation`, such as `spec.java.env`.
- **Existing Go installations:** prefer DDOT SDK for Go `v2.12.0` or later, which supports the shared
  `OTEL_EXPORTER_OTLP_ENDPOINT` for traces ([dd-trace-go#5407][dd-trace-go-5407]). Earlier versions require
  `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=<collector endpoint>/v1/traces` for OTLP/HTTP. Keep that override until the SDK
  is upgraded and shared-endpoint trace delivery is verified.
- **Alternative:** when you don't use compile-time instrumentation with Orchestrion, configure the
  [OpenTelemetry SDK for Go][otel-go] from the standard
  [OpenTelemetry environment variables][otel-sdk-env-vars] instead of hard-coding the configuration in the
  application. The OpenTelemetry Operator's `inject-sdk` annotation, or an equivalent mechanism such as the container
  `env` of [S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator),
  can then inject the configuration. For example:
  - build the resource with `resource.New` and the `resource.WithFromEnv()` option, which reads `OTEL_SERVICE_NAME`
    and `OTEL_RESOURCE_ATTRIBUTES`, or merge it with `resource.Default()`, which includes them;
  - create the exporters with the OTLP exporters' default options, which read the `OTEL_EXPORTER_OTLP_*` variables,
    or with the [`autoexport`][otel-go-autoexport] package, which also reads `OTEL_TRACES_EXPORTER`,
    `OTEL_METRICS_EXPORTER`, and `OTEL_LOGS_EXPORTER`;
  - set the propagators with the [`autoprop`][otel-go-autoprop] package, which reads `OTEL_PROPAGATORS`;
  - don't set an endpoint, protocol, or sampler in the code, because a value set in the code overrides the
    environment variable.

  The DDOT SDK features that the OpenTelemetry SDK for Go doesn't have, such as the `DD_*` features of
  [S-06](#s-06-enable-optional-datadog-features-per-workload), aren't available with this alternative.
- See also the [known gaps](#known-gaps) for Go.

FIXME document how to instantiate the DDOT SDK for Go manually, without Orchestrion, so that it reads its configuration
from the environment variables that the `inject-sdk` annotation or the container `env` of
[S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator) sets, including the `DD_*` variables.
It keeps the Datadog-specific features of the DDOT SDK, which the OpenTelemetry SDK for Go alternative loses: once
documented, present it before that alternative.

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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.env mode=subset sha=56ec68cb9154 -->
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
  - name: OTEL_TRACES_SPAN_METRICS_ENABLED
    value: "false"
  ```

- **Why:** together, these settings enable OpenTelemetry semantics and OTLP export for the signals supported by the
  selected SDK. `DD_TRACE_OTEL_ENABLED=true` alone does not switch trace export from the Datadog protocol to OTLP.
  `OTEL_TRACES_SPAN_METRICS_ENABLED=false` stops the SDK from computing span stats, so the Datadog connector of the
  OpenTelemetry Collector computes the APM trace metrics of every service
  ([C-02](#c-02-compute-apm-trace-metrics-with-the-datadog-connector-before-any-sampling)).
- **Fix:** merge the variables from `instrumentation.env` in [`values.yaml`][values], preserving other list entries.
  Enable traces, metrics, and logs. Install the packages required by the selected SDK for those signals. For logs,
  also choose one log path per workload ([S-10](#s-10-send-each-log-line-to-datadog-once)).
- **Conflicts:** preserve explicit disables, `TracerProvider`, `MeterProvider`, or `LoggerProvider` registrations in
  application code, and alternative exporters until the user approves the migration. Record a signal-specific exception
  when the selected SDK cannot export that signal. Use the [OpenTelemetry API support][dd-otel-api] and
  [OTLP export support matrix][dd-otlp-export] for the selected SDK rather than copying version-specific behavior into
  this guide.

### S-02: Point optional Datadog features at the Datadog Agent

- **Recommendation:** deploy a Datadog Agent alongside the OpenTelemetry setup to enable Datadog-specific
  capabilities. Set `DD_AGENT_HOST` to that reachable Datadog Agent. Allow SDK access to TCP `8126` for features such
  as Remote Configuration, Live Debugger, and Dynamic Instrumentation, and UDP `8125` when using DogStatsD runtime
  metrics ([Datadog Agent network configuration][dd-agent-network]).
- **Without a Datadog Agent:** the basic OTLP path from the SDK to an OpenTelemetry Collector remains available. Preserve an existing
  Datadog Agent destination while a feature still uses it, and avoid sending the same signal through both OpenTelemetry Collector and
  Datadog Agent paths.

### S-03: Use a compatible OTLP transport

- **Requirement:** point each enabled signal at the `daemon` OpenTelemetry Collector endpoint from
  [O-04](#o-04-point-the-sdks-at-the-daemon-opentelemetry-collector). Use the SDK's documented protocol default when the endpoint
  accepts it. Set a shared or signal-specific protocol only when the selected SDK or destination requires an override.
- **Conflicts:** inspect signal-specific endpoint and protocol settings as well as the shared settings. Change a gRPC
  endpoint on port `4317` to HTTP on port `4318` only together with a compatible HTTP protocol. Check the current
  [OTLP export support matrix][dd-otlp-export].

### S-04: Export delta metrics

- **Requirement:** keep the delta temporality preference from `instrumentation.env` in [`values.yaml`][values].

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.env[name=OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE] snippet-path=.[name=OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE] mode=exact sha=6331e98bde06 -->
  ```yaml
  - name: OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE
    value: delta
  ```

- **Why:** `delta` selects delta aggregation temporality for supported instruments, while others remain cumulative as
  required by the OpenTelemetry specification.
- **Conflicts:** when another destination requires cumulative temporality, export cumulative metrics and convert only
  the Datadog branch with the OpenTelemetry Collector's `cumulative_to_delta` processor
  ([C-06](#c-06-convert-cumulative-metrics-to-delta)). Size the OpenTelemetry Collector for the processor's state, set an appropriate
  `max_staleness`, and ensure every point in a time series reaches the same OpenTelemetry Collector instance through the node-local
  Service ([C-13](#c-13-keep-the-daemon-opentelemetry-collectors-service-node-local)).

### S-05: Use the stable semantic conventions

- **Requirement:** keep the stable semantic-convention opt-ins from `instrumentation.env` in
  [`values.yaml`][values].

  ```yaml
  - name: OTEL_SEMCONV_STABILITY_OPT_IN
    value: http,database,rpc,messaging
  ```

### S-06: Enable optional Datadog features per workload

These features of the DDOT SDKs are disabled by default. Enable them in the `Instrumentation` resource's `env` for
every workload, or in a workload's container `env` for that workload only. The commented-out entries of `instrumentation.env`
in [`values.yaml`][values] list them:

| Feature                                    | Variables                                                                              |
|--------------------------------------------|----------------------------------------------------------------------------------------|
| [Dynamic Instrumentation][dd-ddi]          | `DD_DYNAMIC_INSTRUMENTATION_ENABLED=true`, with the Datadog Agent configuration of [A-03](#a-03-enable-live-debugger-and-dynamic-instrumentation) |
| [Database Monitoring][dd-dbm]              | `DD_DBM_PROPAGATION_MODE=service` (other supported values: `full`, `disabled`), `DD_DBM_TRACE_PREPARED_STATEMENTS=true` |
| [Data Streams Monitoring][dd-dsm]          | `DD_DATA_STREAMS_ENABLED=true`, `DD_TRACE_REMOVE_INTEGRATION_SERVICE_NAMES_ENABLED=true` |
| [App & API Protection][dd-aap]             | `DD_APPSEC_ENABLED=true`                                                               |
| [Code Security IAST][dd-iast]              | `DD_IAST_ENABLED=true`                                                                 |
| [Code Security SCA][dd-sca]                | `DD_APPSEC_SCA_ENABLED=true`                                                           |

- **Conflicts:** if the user's workloads already set any of these, keep their values.
- **Requirement:** when migrating a workload from the Datadog SDKs
  ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)), record the Datadog features enabled for it today, and set
  the ones of the table above explicitly in its `env`
  ([S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)). Look in three places:
  - the manifests and the chart values of the workload;
  - the `-Ddd.*` system properties of Java workloads
    ([S-09](#s-09-configure-a-java-agent-bundled-in-the-container-image)), such as `-Ddd.appsec.enabled`;
  - a running pod of the workload, before the switch, which also shows the variables that the Datadog Cluster Agent's
    Admission Controller injects, for example from the `ddTraceConfigs` of a Single Step Instrumentation target or
    from a feature of the Datadog Operator:
    `kubectl get pod <pod> --namespace <namespace> -o yaml | grep -E 'DD_(APPSEC|IAST|PROFILING|DATA_STREAMS|DBM)'`.

  The `admission.datadoghq.com/enabled: "false"` label of S-08 stops these injections: without explicit variables, the
  migrated workload silently loses these features. For `DD_PROFILING_ENABLED`, see the Continuous Profiler item of
  S-13.
- **Precedence:** before changing a variable, find every definition for the workload. The effective order, from highest
  to lowest precedence, is:
  1. the container's `env` in the pod manifest;
  2. the language section of the `Instrumentation` resource, such as `spec.java.env`, for `inject-<language>`;
  3. the `Instrumentation` resource's `spec.env`;
  4. the container's `envFrom` ConfigMaps and Secrets;
  5. the image's `Dockerfile` `ENV` instructions.

  The Operator appends the options that load its SDK to `JAVA_TOOL_OPTIONS`, `NODE_OPTIONS`, and `PYTHONPATH`, and
  merges resource attributes into `OTEL_RESOURCE_ATTRIBUTES`. It skips a language SDK injection when the container sets
  that language's loader variable with `valueFrom`. See the Operator's [SDK injection
  implementation][otel-operator-sdk-go].

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
  `DD_METRICS_OTEL_ENABLED=true` and `OTEL_METRICS_EXPORTER=otlp` effective for that workload.
  `OTEL_METRICS_EXPORTER` defaults to `otlp`: only check that it isn't set to `none`. Check the installed SDK's runtime
  support, required packages or imports, and export transport before applying the change.
- **Java:** from `dd-java-agent` 1.63.0, the JVM runtime metrics go over OTLP when `DD_METRICS_OTEL_ENABLED=true`
  and `OTEL_METRICS_EXPORTER` is `otlp`:
  - they have the OpenTelemetry names, such as `jvm.memory.used`, `jvm.gc.duration`, and `jvm.thread.count`, and the
    `datadog.jvm.runtime` instrumentation scope, in the same OTLP export as the application metrics;
  - JMXFetch stops collecting them, so they don't reach Datadog twice;
  - `DD_RUNTIME_METRICS_ENABLED` defaults to `true` in Java: you don't need to set it.

  Before 1.63.0, the JVM runtime metrics go through DogStatsD. The JMXFetch integrations, such as the Tomcat or
  Kafka JMX metrics, and custom JMXFetch configurations, always go through DogStatsD: keep UDP `8125` of the Datadog
  Agent reachable for them, or report them as `blocked`. Datadog maps the OpenTelemetry names for the service page and
  the default runtime dashboards ([OpenTelemetry runtime metrics][dd-otel-runtime-metrics]).
- **Compatibility:** follow the [runtime metrics documentation][dd-runtime-metrics]. If the SDK sends runtime metrics
  only through DogStatsD, keep the existing reachable Datadog Agent path. For a setup without a Datadog Agent, report
  runtime-metrics delivery as `blocked` until a supported delivery path is configured.
- **Verify:** confirm that runtime metrics arrive in Datadog with the expected service and environment tags through the
  intended transport. An enabled flag or successful trace export is insufficient.
- **Conflicts:** preserve explicit runtime-metrics disables and existing runtime instrumentation. Ask before enabling disabled
  runtime-metrics collection or adding instrumentation that could duplicate the same measurements.

FIXME check in the source code of the other DDOT SDKs (Python, Node.js, Go, .NET, Ruby, PHP) whether runtime
metrics go over OTLP or DogStatsD in the OpenTelemetry mode, from which version, with which metric names, and whether
`DD_RUNTIME_METRICS_ENABLED` is needed. Then add a per-language item, as for Java. Also check which sampler they use in
OTLP mode without sampling settings, whether they only export the sampled traces with their probability in the
`tracestate`, as `dd-java-agent` does ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)), and whether the
`datadog` connector of the upstream OpenTelemetry Collector Contrib scales the trace metrics with it.

### S-08: Configure the SDKs of workloads without the OpenTelemetry Operator

- **Requirement:** when the OpenTelemetry Operator doesn't inject the OpenTelemetry configuration into a workload,
  add these variables to the `env` of the application container, replacing the `<...>` placeholders. This is
  typically the case when the OpenTelemetry SDKs are managed manually: the SDK is included in the container image of
  the service, and its OpenTelemetry configuration is managed in the Kubernetes deployment. The variables form the
  following groups:
  - **OpenTelemetry SDK configuration**, for all OpenTelemetry SDKs: the `daemon` collector endpoint
    ([O-04](#o-04-point-the-sdks-at-the-daemon-opentelemetry-collector)), the delta temporality
    ([S-04](#s-04-export-delta-metrics)), and the stable semantic conventions
    ([S-05](#s-05-use-the-stable-semantic-conventions)). The SDK's default OTLP protocol applies: only set
    `OTEL_EXPORTER_OTLP_PROTOCOL` when the SDK requires an override ([S-03](#s-03-use-a-compatible-otlp-transport)).
  - **Resource attributes**, for all OpenTelemetry SDKs, following the
    [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes].
  - **DDOT SDK configuration**, only for the DDOT SDKs: the OpenTelemetry mode
    ([S-01](#s-01-enable-the-opentelemetry-mode-of-the-ddot-sdks)), the Datadog Agent host
    ([S-02](#s-02-point-optional-datadog-features-at-the-datadog-agent)), and `OTEL_TRACES_SPAN_METRICS_ENABLED=false`, which
    leaves the computation of the APM trace metrics to the Datadog connector of the OpenTelemetry Collector
    ([C-02](#c-02-compute-apm-trace-metrics-with-the-datadog-connector-before-any-sampling)). To also enable optional Datadog features, such as
    Dynamic Instrumentation, Database Monitoring, Data Streams Monitoring, App & API Protection, and Code Security,
    add their variables to this group ([S-06](#s-06-enable-optional-datadog-features-per-workload)).

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.env mode=adapted sha=56ec68cb9154 -->
  ```yaml
  spec:
    containers:
      - env:
          ## OpenTelemetry SDK configuration
          - name: OTEL_EXPORTER_OTLP_ENDPOINT
            value: http://<helm-release-name>-daemon-collector.<namespace>.svc.cluster.local:4318
          - name: OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE
            value: delta
          - name: OTEL_SEMCONV_STABILITY_OPT_IN
            value: http,database,rpc,messaging

          ## Resource attributes to implement https://opentelemetry.io/docs/specs/semconv/non-normative/k8s-attributes/
          - name: OTEL_SERVICE_NAME
            value: <service name>
          # Only when the workload has a Datadog env, tags.datadoghq.com/env or DD_ENV, with its value (S-11), when
          # the OpenTelemetry Collector can't set deployment.environment.name, or when a configuration management
          # solution, such as the OpenTelemetry Kube Stack Helm chart, sets it in both the OpenTelemetry Collector and
          # the SDKs, with the same value (C-11). When you remove this variable, also remove the
          # deployment.environment.name entry of OTEL_RESOURCE_ATTRIBUTES below.
          - name: OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT_NAME
            value: <deployment environment name>
          - name: OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION
            value: <service version>
          # Must equal this container's name in the pod spec: the downward API can't read it.
          - name: OTEL_RESOURCE_ATTRIBUTES_CONTAINER_NAME
            value: <name of this container>
          - name: OTEL_RESOURCE_ATTRIBUTES_POD_NAME
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.name
          - name: OTEL_RESOURCE_ATTRIBUTES_POD_UID
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.uid
          - name: OTEL_RESOURCE_ATTRIBUTES_NODE_NAME
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: spec.nodeName
          - name: OTEL_RESOURCE_ATTRIBUTES_POD_NAMESPACE
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.namespace
          - name: OTEL_RESOURCE_ATTRIBUTES
            value: deployment.environment.name=$(OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT_NAME),k8s.namespace.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAMESPACE),k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),k8s.pod.uid=$(OTEL_RESOURCE_ATTRIBUTES_POD_UID),k8s.container.name=$(OTEL_RESOURCE_ATTRIBUTES_CONTAINER_NAME),service.instance.id=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAMESPACE).$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).$(OTEL_RESOURCE_ATTRIBUTES_CONTAINER_NAME),service.version=$(OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION)

          ## DDOT SDK configuration, only for the DDOT SDKs
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
          - name: DD_AGENT_HOST
            value: <Datadog Agent Service DNS name, typically datadog-agent.datadog.svc.cluster.local>
          # Let the Collector compute the span stats, so every service gets trace.* metrics
          - name: OTEL_TRACES_SPAN_METRICS_ENABLED
            value: "false"
  ```

- **Requirement:** when no workload of the cluster uses the injection of the OpenTelemetry Operator, for example when
  the SDKs are in the images, set `instrumentation.enabled: false` in the values of the OpenTelemetry Kube Stack Helm
  chart: the chart then doesn't create the `Instrumentation` resource, which no workload needs. The OpenTelemetry
  Operator still deploys the OpenTelemetry Collectors.
- **Requirement:** add the `admission.datadoghq.com/enabled: "false"` label to the pod template of the workload, as for
  the workloads that the OpenTelemetry Operator instruments
  ([O-05](#o-05-annotate-workloads-and-prevent-double-instrumentation)):

  ```yaml
  spec:
    template:
      metadata:
        labels:
          admission.datadoghq.com/enabled: "false" # label values are strings: keep the quotes
  ```

  The label turns off all the mutations of the Datadog Cluster Agent's Admission Controller for the pods of the
  workload. It no longer injects:
  - a Datadog SDK with Single Step Instrumentation;
  - the `DD_SERVICE`, `DD_VERSION`, and `DD_ENV` variables, which can take precedence over the OpenTelemetry service
    identity ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity));
  - the path to the Datadog Agent: `DD_AGENT_HOST`, in its `hostip` and `service` modes, or the
    `DD_TRACE_AGENT_URL` and `DD_DOGSTATSD_URL` variables and the volumes of the Datadog Agent's sockets, in its
    `socket` and `csi` modes. The `DD_AGENT_HOST` of the variables above replaces them with the local Service of the
    Datadog Agent: this is tested with the default configuration of the Datadog Operator, where the pods used the
    sockets before, for Live Debugger, Feature Flag Management, Data Streams Monitoring, and App & API Protection;
  - the `DD_ENTITY_ID`, `DD_EXTERNAL_ENV`, and `DD_INTERNAL_POD_UID` variables, which the Datadog Agent uses for the
    origin detection of what the SDK sends to it. The telemetry that the SDK exports over OTLP gets its Kubernetes
    attributes from the variables above. When a migrated workload still sends DogStatsD metrics to the Datadog Agent,
    for example the JMXFetch integrations of [S-07](#s-07-enable-runtime-metrics-where-supported), add them with the
    downward API, as the Admission Controller does: `DD_ENTITY_ID` from `metadata.uid`, and `DD_EXTERNAL_ENV` with
    the value `it-false,cn-<name of this container>,pu-$(OTEL_RESOURCE_ATTRIBUTES_POD_UID)`, after
    `OTEL_RESOURCE_ATTRIBUTES_POD_UID`.
- **Requirement:** in addition to the resource attribute variables above, which only configure the SDK, annotate
  the pod template with the `resource.opentelemetry.io/*` annotations, such as
  `resource.opentelemetry.io/service.name` and `resource.opentelemetry.io/service.version`
  ([C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs)). Use the same values as in the
  environment variables. The OpenTelemetry Collector produces telemetry about the same pod, such as its logs,
  metrics, and Kubernetes events, and can't read the SDK's environment variables. With the `k8s_attributes`
  processor and `extract.otel_annotations: true`, the Collector reads the annotations and enriches this telemetry
  with the same resource attributes, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes].
  Add the annotations to the pod template of the workload, for example of a Deployment:

  ```yaml
  spec:
    template:
      metadata:
        annotations:
          resource.opentelemetry.io/service.name: <service name, the value of OTEL_SERVICE_NAME>
          resource.opentelemetry.io/service.version: <service version, the value of OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION>
          # Only when the workload has a Datadog env, with its value (S-11), when the OpenTelemetry Collector can't
          # set deployment.environment.name, or when a configuration management solution, such as the OpenTelemetry
          # Kube Stack Helm chart, sets it in both the OpenTelemetry Collector and the SDKs, with the same value (C-11)
          resource.opentelemetry.io/deployment.environment.name: <deployment environment name, the value of OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT_NAME>
  ```

  Update `resource.opentelemetry.io/service.version` together with `OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION` on every
  release. Don't set the `resource.opentelemetry.io/service.instance.id` annotation: the environment variables already
  compute `service.instance.id` with the conventions' default, `<namespace>.<pod>.<container>`. With the annotation,
  the OpenTelemetry Collector would use its value instead, which can differ from the SDK's.
- **Why:** on Kubernetes, the OpenTelemetry Operator sets the resource attributes of the SDKs it configures, following
  the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]. Without the Operator, the resource
  attribute variables above and the pod annotations follow the same conventions. These resource attributes serve the
  following goals:
  - **Correlation with the rest of the telemetry:** the SDKs' telemetry must carry the same resource attributes as the
    telemetry that other components collect about the same workload. These components include the OpenTelemetry
    Collector, for pod logs, metrics, and Kubernetes events, the OpenTelemetry eBPF profiler, and OpenTelemetry eBPF
    instrumentation. The attributes that link them include `service.name`, `service.namespace`,
    `service.instance.id`, `k8s.pod.name`, `k8s.pod.uid`, `k8s.deployment.name`, `k8s.cluster.name`, and
    `k8s.node.name`. With `k8s.pod.uid`, the `k8s_attributes` processor associates the telemetry with its pod, and with
    `k8s.container.name`, it finds the container in pods that have several containers.
  - **`service.instance.id` in the environment variables:** keep it in `OTEL_RESOURCE_ATTRIBUTES`, even though the
    `k8s_attributes` processor can compute the same `<namespace>.<pod>.<container>` value. Some SDKs generate a random
    `service.instance.id` when none is set, for example the upstream OpenTelemetry Java agent, and the processor
    doesn't overwrite an attribute that the telemetry already has. The SDK's telemetry would then keep a random ID
    that changes on every restart and doesn't match the telemetry the OpenTelemetry Collector produces about the pod.
    The processor also can't compute the value without `k8s.container.name` or `container.id`, for example in pods
    that have several containers.
  - **Organization of the telemetry:** `service.name`, `service.namespace`, `deployment.environment.name`, and the
    custom resource attributes that classify services let you group and filter the telemetry, for example by
    application and environment.
  - **APM troubleshooting:** some APM capabilities depend on specific attributes. For example, `service.version`
    enables [Deployment Tracking][dd-deployment-tracking], which compares the behavior of a service across its
    versions.
- **`deployment.environment.name`:** setting it in the OpenTelemetry Collector's ingestion pipeline is strongly
  recommended. Only keep `OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT_NAME`, the `deployment.environment.name`
  entry of `OTEL_RESOURCE_ATTRIBUTES`, and the `resource.opentelemetry.io/deployment.environment.name` annotation
  when the workload has a Datadog environment, `tags.datadoghq.com/env` or `DD_ENV`
  ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)), when the
  OpenTelemetry Collector can't set it, or when one configuration automation sets it in both the OpenTelemetry
  Collector and the SDKs ([C-11](#c-11-set-the-deployment-environment)).
- **Fix:** keep `OTEL_RESOURCE_ATTRIBUTES` after the variables it references. Kubernetes only expands a `$(VAR)`
  reference to a variable defined earlier in the same container's `env`, and leaves the other references as literal
  text ([dependent environment variables][k8s-dependent-env-vars]). For Go workloads that use a DDOT SDK for Go
  version earlier than `v2.12.0`, also set `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`
  ([O-06](#o-06-instrument-go-applications-at-compile-time)). With the upstream OpenTelemetry SDKs, leave out the
  DDOT SDK configuration group.
- **With a shared Helm chart:** when the services are deployed with a shared, internal Helm chart, add the variables
  and the annotations above to the pod template of the chart, once for all the services. A new version of the chart
  then rolls the configuration out to all of them, through the existing deployment or GitOps pipeline.
  - Take the per-service values from the values that the chart already has, and compute each one once, for example
    in a named template such as `{{ include "<chart>.serviceName" . }}`, so that the labels, the variables, and the
    annotations always match:
    - when the chart already sets the Datadog Unified Service Tagging values, the `tags.datadoghq.com/*` labels or
      the `DD_SERVICE`, `DD_VERSION`, and `DD_ENV` variables, use these values for the service name, the service
      version, and the deployment environment
      ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity));
    - otherwise, for the services that already report to Datadog APM, the service name and the environment in effect
      today, read from a running pod
      ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)), added to the
      values of each service;
    - otherwise, for the services that don't report to Datadog APM yet, the service name from the chart's service or
      release name;
    - the service version from the image tag that the chart deploys, when the chart has no Datadog version value. The
      service version then changes on every release without other updates;
    - the container name from the value that the template uses for `containers[].name`, so that it can't drift from
      the pod spec.
  - When the chart has no Datadog environment value, leave `deployment.environment.name` out of the chart, so that the
    same values serve all the environments ([C-11](#c-11-set-the-deployment-environment)). Use a values file per
    environment only when the OpenTelemetry Collector can't set it.
  - Make the `daemon` OpenTelemetry Collector endpoint and `DD_AGENT_HOST` chart values with defaults, and let
    services enable the optional Datadog features ([S-06](#s-06-enable-optional-datadog-features-per-workload)) and
    choose their log path ([S-10](#s-10-send-each-log-line-to-datadog-once)) with values.
  - **Switch each service separately,** to migrate the services one at a time
    ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)), with two boolean values that each service sets in its own
    values, off by default during the migration:
    - `openTelemetry.apmSdk.enabled` adds the environment variables above, and the
      `admission.datadoghq.com/enabled: "false"` label. The services that aren't migrated yet keep the Datadog SDKs in
      their default mode, and the variables that the Datadog Cluster Agent's Admission Controller injects.
    - `openTelemetry.podLogs.enabled` adds the `ad.datadoghq.com/logs_exclude: "true"` annotation, when the `daemon`
      OpenTelemetry Collector tails the pod logs instead of the Datadog Agent
      ([C-14](#c-14-choose-where-the-opentelemetry-collector-runs-on-kubernetes) level 1). A service can then move
      its pod logs after its APM SDK, for example once its log pipelines are checked. At levels 2 and 3, the Datadog
      Agent keeps tailing the pod logs, and this switch isn't needed.

    Keep the `resource.opentelemetry.io/*` annotations on for all the services: they're harmless before the
    migration, and the OpenTelemetry Collector can use them for the pod logs and metrics of every service.

    **At the end of the migration,** some services can stay on the Datadog SDK on purpose, for example the ones that
    need memory or lock profiling ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)), or whose monitors aren't
    ready. Turning the switches on by default would move them silently. So, before changing the default:
    1. list the services that don't set the switches in their own values, and rely on the default;
    2. set an explicit value for each of them: `true` for the migrated ones, `false` for the ones that stay on the
       Datadog SDK;
    3. turn the switches on by default, so that new services start with the DDOT SDK.

    Remove the switches only when no service sets them to `false` anymore.

    `openTelemetry.apmSdk.enabled` switches the traces, metrics, and logs of the SDK, and the
    `admission.datadoghq.com/enabled: "false"` label, at once. Start with one canary service, in a non-production
    environment first: turn the switch on, check its traces, metrics, and logs in Datadog
    ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks) verify), then switch the other services in small groups.
    Check that the OTLP endpoint of the chart is reachable from the namespace of each service before turning its
    switch on, as step 2 of the [instructions for coding agents](#instructions-for-coding-agents) shows: without it,
    the service stops sending traces. With GitOps, for example Argo CD or Flux, set the switches in the values of each
    service in Git, and change them only there, also to roll back
    ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)).

  For example, with value names to map to the ones of your chart:

  ```yaml
  spec:
    template:
      metadata:
        annotations:
          resource.opentelemetry.io/service.name: {{ .Values.service.name | quote }}
          resource.opentelemetry.io/service.version: {{ .Values.image.tag | quote }}
          {{- if .Values.openTelemetry.podLogs.enabled }}
          ad.datadoghq.com/logs_exclude: "true"
          {{- end }}
        {{- if .Values.openTelemetry.apmSdk.enabled }}
        labels:
          admission.datadoghq.com/enabled: "false"
        {{- end }}
      spec:
        containers:
          - name: {{ .Values.service.name }}
            env:
              {{- if .Values.openTelemetry.apmSdk.enabled }}
              - name: OTEL_SERVICE_NAME
                value: {{ .Values.service.name | quote }}
              - name: OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION
                value: {{ .Values.image.tag | quote }}
              - name: OTEL_RESOURCE_ATTRIBUTES_CONTAINER_NAME
                value: {{ .Values.service.name | quote }}   # the value of containers[].name
              # ... the other variables of the snippet above
              {{- end }}
  ```

  If the chart already defines `labels` in the pod template, add the label there instead of in a second `labels` key.
  If it already sets `admission.datadoghq.com/enabled: "true"`, which the Datadog Cluster Agent's Admission Controller
  requires to mutate a pod when its `mutateUnlabelled` setting is `false`, the default of the Datadog Operator and the
  `datadog` Helm chart, set the value of that label from the switch instead of adding a second key, for example
  `admission.datadoghq.com/enabled: {{ ternary "false" "true" .Values.openTelemetry.apmSdk.enabled | quote }}`. With
  two keys in the same map, the rendering fails, or keeps only one of them, depending on the tool.

- **Conflicts:** if the container already sets `OTEL_RESOURCE_ATTRIBUTES`, merge its attributes into the new value
  and keep the user's values when they differ. For the other variables, follow the conflicts of the rules each group
  links to, such as [S-01](#s-01-enable-the-opentelemetry-mode-of-the-ddot-sdks) for explicitly disabled signals.

### S-09: Configure a Java agent bundled in the container image

- **Requirement:** when the container image already contains the Datadog `dd-trace-java` agent, `dd-java-agent`, for
  example in a base image that a platform team maintains, keep it as the DDOT SDK Java agent, and make sure that the
  JVM loads exactly one Java agent with `-javaagent`:
  - with the OpenTelemetry Operator, use the `instrumentation.opentelemetry.io/inject-sdk` annotation. It injects the
    configuration of the `Instrumentation`, including the `DD_*` variables, without injecting a Java agent
    ([O-06](#o-06-instrument-go-applications-at-compile-time)). Don't use `inject-java`: the Operator would add the
    DDOT SDK Java agent of its image to `JAVA_TOOL_OPTIONS`, and the JVM would load two Java agents;
  - without the OpenTelemetry Operator, set the variables in the container `env`
    ([S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)).
- **Requirement:** don't expand `JAVA_TOOL_OPTIONS` on the `java` command line in the startup scripts, for example in
  `exec java ${JAVA_OPTS} ${JAVA_TOOL_OPTIONS} ...`: the JVM already reads `JAVA_TOOL_OPTIONS` from the environment, so
  its options apply twice. A `-javaagent` that a mechanism adds to `JAVA_TOOL_OPTIONS`, such as the OpenTelemetry
  Operator's `inject-java` annotation or the Single Step Instrumentation of the Datadog Cluster Agent's Admission
  Controller, then loads twice. Remove the expansion, and keep the other options in a separate variable.
  Removing it can change which option wins. The JVM reads `JAVA_TOOL_OPTIONS` before the command line, and the last
  occurrence of an option wins. In `exec java ${JAVA_OPTS} ${JAVA_TOOL_OPTIONS} ...`, the `JAVA_TOOL_OPTIONS` options
  come last and win over the ones of `JAVA_OPTS`. Without the expansion, the options of `JAVA_OPTS` win, for example a
  different `-XX:MaxRAMPercentage` or `-Ddd.*` value. Before the change, compare the options that both variables set
  in each workload, and remove the duplicates, keeping the value that's effective today.
- **Requirement:** use `dd-java-agent` 1.68.0 or later, the first version that supports all the settings of
  [S-01](#s-01-enable-the-opentelemetry-mode-of-the-ddot-sdks). Otherwise, upgrade `dd-java-agent` in the image, and
  pin its version. The minimums per feature are:

  | Feature                                                                       | Minimum `dd-java-agent` version |
  |-------------------------------------------------------------------------------|---------------------------------|
  | OTLP trace export                                                             | 1.62.0                          |
  | OpenTelemetry Metrics API                                                     | 1.61.0                          |
  | OpenTelemetry Logs API, with `opentelemetry-api` 1.27.0 or later              | 1.62.0                          |
  | JVM runtime metrics over OTLP                                                 | 1.63.0                          |
  | Sampling probability in OTLP spans, for the trace metrics                     | 1.67.0                          |
  | OpenTelemetry semantic conventions of HTTP, `DD_TRACE_OTEL_SEMANTICS_ENABLED` | 1.68.0                          |
  | Metrics shutdown in short-lived processes                                     | 1.68.0                          |

  These versions come from the [OTLP export support matrix][dd-otlp-export] and the
  [OpenTelemetry API support][dd-otel-api] pages, which are authoritative when they change. The `datadog` connector
  uses the sampling probability of the OTLP spans from Datadog Agent 7.82.0
  ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)).
- **Requirement:** find every `-Ddd.*` system property of the JVM, and remove or align the ones that configure the
  same settings as an environment variable of this guide, for example `-Ddd.logs.injection`
  ([S-10](#s-10-send-each-log-line-to-datadog-once)) or `-Ddd.trace.propagation.style`
  ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)). In the Java tracer, a system property takes priority
  over the environment variable of the same setting, for example `-Ddd.trace.otel.enabled` over
  `DD_TRACE_OTEL_ENABLED` ([Java tracer configuration][dd-java-config]). A leftover system property can then silently
  cancel an environment variable. Look for them:
  - in `JAVA_TOOL_OPTIONS` and `JDK_JAVA_OPTIONS`;
  - in custom variables that a startup script expands into the `java` command, for example a `JAVA_OPTS` variable;
  - in the `ENTRYPOINT` and `CMD` of the image, and in its startup scripts;
  - in a properties file that `DD_TRACE_CONFIG` points to.

  To convert a system property to its environment variable, uppercase it and replace `.` and `-` with `_`, for
  example `dd.logs.injection` becomes `DD_LOGS_INJECTION`. Also consider moving the `-Ddd.*` properties that don't
  conflict, such as `-Ddd.logs.injection=true`, to their `DD_*` environment variables, so that the configuration of
  `dd-java-agent` is in one place, as the
  [application workflow](#3-move-compatible-configuration-to-environment-variables) recommends. When you move them:
  - put the variables in the container `env` of all the workloads that had the property, not only behind the switch
    of the DDOT SDK configuration ([S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)),
    so that the workloads still on the Datadog SDK keep the setting;
  - check that no other definition sets the same variable with another value, such as the chart, an `envFrom`, the
    image, or the Datadog Cluster Agent's Admission Controller: unlike the system property, the variable doesn't win
    over them ([S-06](#s-06-enable-optional-datadog-features-per-workload) precedence);
  - don't carry `-Ddd.profiling.enabled=true` into the configuration of the DDOT SDK, which doesn't support
    in-process profiling ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)): remove it from the workloads that
    move to the DDOT SDK, and keep it on the ones that stay on the Datadog SDK.

  Keep the JVM options
  that don't configure the telemetry, such as memory settings, and the `-Ddatadog.*` properties, which configure the
  tracer's own logging: this guide doesn't require changing them. With
  `-Ddatadog.slf4j.simpleLogger.logFile=System.out`, or with the default, the standard error, the tracer's logs reach
  Datadog with the application's logs that the OpenTelemetry Collector or the Datadog Agent tails
  ([S-10](#s-10-send-each-log-line-to-datadog-once)). At the default log level, they're few, such as the startup
  configuration and warnings, and they help troubleshooting. Only when they disturb the log parsing, send them to a
  file in a writable path, for example `-Ddatadog.slf4j.simpleLogger.logFile=/tmp/dd-java-agent.log`, and read it with
  `kubectl exec`. The variable that holds the option, for example in the workload's `env`, decides who makes the
  change.
- **Logs:** `dd-java-agent` doesn't export the logs of Log4j 2, Logback, or `java.util.logging` over OTLP: tail the
  standard output of these applications ([S-10](#s-10-send-each-log-line-to-datadog-once)).
- **Verify:** check the effective JVM arguments and environment of a running pod, then the active configuration in
  the **SDK Configuration** tab of the service's **Service Config** in Datadog APM:

  ```sh
  kubectl exec <pod> --namespace <namespace> -- sh -c 'tr "\0" " " < /proc/1/cmdline; echo; env | grep -E "^(DD|OTEL|JAVA_TOOL)_"'
  ```

  `/proc/1/cmdline` shows the `java` command when the startup script starts it with `exec`. Otherwise, find the
  process ID of `java` first. The command line doesn't show the options that the JVM reads from `JAVA_TOOL_OPTIONS`:
  look for them in the JVM's `Picked up JAVA_TOOL_OPTIONS:` message, and count the `-javaagent` options of both the
  command line and that message, which must total one:

  ```sh
  kubectl logs <pod> --namespace <namespace> | grep 'Picked up JAVA_TOOL_OPTIONS'
  ```
- **Conflicts:** if the Datadog Cluster Agent's Admission Controller or another mechanism also adds a `-javaagent`, keep
  only one ([O-05](#o-05-annotate-workloads-and-prevent-double-instrumentation)).

### S-10: Send each log line to Datadog once

- **Requirement:** each workload's logs reach Datadog through one path: either the `daemon` OpenTelemetry Collector or
  the Datadog Agent tailing the container's standard output and error, or OTLP export from the SDK. The reference
  configuration does both by default: the `file_log` receiver of the `daemon` OpenTelemetry Collector tails the logs of
  every pod, with `include: [/var/log/pods/*/*/*.log]` and no `exclude`, and S-01 sets `DD_LOGS_OTEL_ENABLED=true`. Both
  paths feed the same `logs` pipeline. When the SDK also exports the logs of the application's logging library over
  OTLP, which the Python and .NET DDOT SDKs do (see the table below), every log line that the application also prints
  reaches Datadog twice.
- **Requirement:** inspect the existing log ownership before changing it. Render the configuration of every
  OpenTelemetry Collector and Datadog Agent that can collect the workload's logs. Record the matching
  `file_log.include` and `file_log.exclude` patterns. When an existing receiver already matches the workload, adapt
  that receiver instead of adding another one.
- **Fix:** choose one path per workload:
  - **Tail the standard output,** when the application already logs to it. This path collects everything the
    container prints, including the output of libraries that the SDK doesn't instrument. When the SDK exports the
    events of the application's logging library over OTLP, set `DD_LOGS_OTEL_ENABLED=false` in the workload's
    container `env`, which takes precedence over the `Instrumentation`
    ([S-06](#s-06-enable-optional-datadog-features-per-workload)): see the table below. Keep log injection,
    `DD_LOGS_INJECTION=true`, so that the log lines carry the trace and span IDs, and prefer JSON logs.
  - **Export the logs over OTLP.** The log records carry the trace context and the resource attributes of the SDK,
    without parsing. Only the log events of the logging libraries that the SDK supports are exported. Exclude the
    container's log files from `file_log`, with a pattern of the form
    `/var/log/pods/<namespace>_<pod name prefix>*_*/<container name>/*.log` in its `exclude` list.
- **`DD_LOGS_OTEL_ENABLED=true` per language:** it always exports the logs that the application emits with the
  OpenTelemetry Logs API ([OpenTelemetry API support][dd-otel-api]). Only some DDOT SDKs also capture the
  application's usual logging library, in parallel with its existing output, such as the standard output. Their log
  lines then reach Datadog twice when that output is also tailed:

  | DDOT SDK | Logging library captured with `DD_LOGS_OTEL_ENABLED=true` | When the standard output is tailed |
  |----------|-----------------------------------------------------------|------------------------------------|
  | Python   | Python's `logging` module: the SDK attaches an OpenTelemetry `LoggingHandler` to the root logger | Set `DD_LOGS_OTEL_ENABLED=false`, or export over OTLP and stop tailing |
  | .NET     | `ILogger` (`Microsoft.Extensions.Logging`): the SDK adds a logger provider to the application's `LoggerFactory`. Not Serilog, NLog, or log4net | Same as Python for `ILogger` |
  | Java     | None: not Log4j 2, Logback, or `java.util.logging` | Keep `DD_LOGS_OTEL_ENABLED=true` |
  | Node.js  | None: not winston, pino, or bunyan | Keep `DD_LOGS_OTEL_ENABLED=true` |
  | Go       | None: logs are only exported after the application sets up the OpenTelemetry logger provider | Keep `DD_LOGS_OTEL_ENABLED=true` |
  | Ruby     | None: not Ruby's `Logger` | Keep `DD_LOGS_OTEL_ENABLED=true` |
  | PHP      | Monolog, only when the application attaches the OpenTelemetry Monolog handler to a logger | Keep `DD_LOGS_OTEL_ENABLED=true`, unless that handler and a standard output handler are both attached |

  For example, a Java application that logs with Log4j 2 to the standard output keeps tailing it, with
  `DD_LOGS_INJECTION=true`, and keeps `DD_LOGS_OTEL_ENABLED=true` for the logs it emits with the OpenTelemetry Logs
  API. Check that the log lines still carry the trace and span IDs. The Python, .NET, Java, and Node.js rows are read
  from the source code of the Datadog SDKs; the Go, Ruby, and PHP rows come from the
  [OpenTelemetry API support][dd-otel-api] page.
- **Conflicts:** don't disable `file_log` for all the pods: it also collects the logs of the workloads that don't
  export logs over OTLP. Record the path chosen for each workload.
- **Conflicts:** the `ad.datadoghq.com/logs_exclude: "true"` annotation only stops the Datadog Agent from collecting
  the container logs. OpenTelemetry `file_log` receivers don't read this annotation. To stop an OpenTelemetry
  Collector from tailing a workload, change that receiver's `include` or `exclude` patterns.
- **Verify:** in the Datadog Log Explorer, a log line of the workload appears once, and its logs link to its traces.

### S-11: Align Datadog Unified Service Tagging with the OpenTelemetry service identity

- **Requirement:** when workloads have the [Unified Service Tagging][dd-unified-service-tagging] labels
  `tags.datadoghq.com/service`, `tags.datadoghq.com/version`, and `tags.datadoghq.com/env`, on the pod template or the
  parent workload, the OpenTelemetry service identity of the workload has the same values: `service.name`,
  `service.version`, and `deployment.environment.name`. Pods with several containers can have per-container labels,
  such as `tags.datadoghq.com/<container name>.service`.
- **Why:** these labels set more than the SDK configuration:
  - the Datadog Agent tags the telemetry it collects about the containers, such as logs and metrics, from them;
  - the Datadog Cluster Agent's Admission Controller injects `DD_SERVICE`, `DD_VERSION`, and `DD_ENV` from them in
    the pods it mutates, and manifests often set the same variables from the labels with the downward API;
  - in the DDOT SDKs, these `DD_*` variables can take precedence over the OpenTelemetry values
    ([OpenTelemetry environment variable interoperability][dd-otel-env]).

  When the values differ, the traces, logs, and metrics of a workload get different `service`, `version`, or `env`
  tags, and Datadog can't correlate them.
- **Requirement, strongly recommended:** when a workload already has a Datadog value, a `tags.datadoghq.com/*` label
  or a `DD_SERVICE`, `DD_VERSION`, or `DD_ENV` variable, that value is the source of the OpenTelemetry service
  identity. Set it, with the same value, in both:
  - the SDK variables of [S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator), which
    configure the SDK: `OTEL_SERVICE_NAME`, and the `service.version` and `deployment.environment.name` entries of
    `OTEL_RESOURCE_ATTRIBUTES`. The `admission.datadoghq.com/enabled: "false"` label of S-08 stops the Datadog Cluster
    Agent's Admission Controller from injecting the `DD_*` variables from the labels, so the SDK only gets these
    values from the variables you set;
  - the `resource.opentelemetry.io/*` pod annotations, which the `k8s_attributes` processor reads for the telemetry
    that the OpenTelemetry Collector collects about the pod, such as its logs, metrics, and Kubernetes events
    ([C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs)).

  | Datadog source                                 | OpenTelemetry attribute       | SDK variable                                                                   | Pod annotation                                         |
  |------------------------------------------------|-------------------------------|--------------------------------------------------------------------------------|--------------------------------------------------------|
  | `tags.datadoghq.com/service` or `DD_SERVICE`   | `service.name`                | `OTEL_SERVICE_NAME`                                                            | `resource.opentelemetry.io/service.name`               |
  | `tags.datadoghq.com/version` or `DD_VERSION`   | `service.version`             | `OTEL_RESOURCE_ATTRIBUTES`, through `OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION` | `resource.opentelemetry.io/service.version`            |
  | `tags.datadoghq.com/env` or `DD_ENV`           | `deployment.environment.name` | `OTEL_RESOURCE_ATTRIBUTES`, through `OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT_NAME` | `resource.opentelemetry.io/deployment.environment.name` |

  This keeps the existing Datadog service name, so the APM service, its monitors, and its history continue after the
  migration ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)).
- **Requirement:** when the configuration has no Datadog value but the workload already reports to Datadog APM, the
  values in effect today are the source of `service.name` and `deployment.environment.name`. The Datadog SDK can
  derive a service name itself, for example from the jar or the main class in Java, the Datadog Cluster Agent's
  Admission Controller can inject `DD_SERVICE` from the workload name, and the Datadog Agent can add its own `env`.
  Nothing in the manifests or the chart shows these values, and the `admission.datadoghq.com/enabled: "false"` label
  of S-08 stops the injection. Read them from a running pod, before the switch:
  - in the configuration line that the Datadog SDK logs at startup, for example `DATADOG TRACER CONFIGURATION` in
    Java, which has the `service`, `env`, and `version` in effect:
    `kubectl logs <pod> --namespace <namespace> | grep 'DATADOG TRACER CONFIGURATION'`;
  - or in Datadog APM, in the tags of the service, or in the **SDK Configuration** tab of its **Service Config**.
  - when the startup line has no `env` but Datadog APM shows one, in the default `env` of the Datadog Agent. The
    Datadog Agent applies it to the traces that arrive without one, from its `apm_config.env` or `env` settings, or
    an `env:` entry of its `tags`, such as `DD_ENV`, `DD_TAGS`, or `spec.global.tags` of the `DatadogAgent`:
    `kubectl exec <Datadog Agent pod> --namespace <namespace> -c agent -- agent config | grep -E '^ *env:|^tags:'`.
    This default is a value for the whole cluster: [C-11](#c-11-set-the-deployment-environment) says where to set it
    after the switch.

  Then pin them, as for a Datadog value: in the `tags.datadoghq.com/*` labels, the SDK variables, and the
  annotations, with the same value. For `service.version`, which changes on every release, the image tag of S-08
  is enough. Use the other sources of S-08, such as the chart's service name, only for the workloads that don't report
  to Datadog APM yet. Set `deployment.environment.name` in the OpenTelemetry Collector
  ([C-11](#c-11-set-the-deployment-environment)) when no `env` is in effect, or when the `env` in effect is the
  default of the Datadog Agent.
- **Requirement:** don't change the `service` or `env` of a workload that reports to Datadog APM without the user's
  approval. Report each change as a decision, with the value before and after.
- **Requirement:** when the workload carries `deployment.environment.name`, the OpenTelemetry Collector doesn't
  overwrite it. Don't set `deployment.environment.name` in the `resourceAttributes` of the OpenTelemetry Kube Stack
  Helm chart, which uses `action: upsert`. To keep a default for the other workloads, use a `resource` processor with
  `action: insert`, after `k8s_attributes`, as the conflicts of [C-11](#c-11-set-the-deployment-environment) describe.
  The same applies to the bundled DDOT Collector
  ([S-12](#s-12-send-otlp-to-the-datadog-agent-instead-of-standalone-opentelemetry-collectors)). The OpenTelemetry
  Collectors don't set `service.name` or `service.version`, and the `infraattributes` processor only adds the values
  that the telemetry doesn't have.
- **Troubleshooting:** when SDK telemetry has the expected `env` but pod logs have the Collector's cluster-wide
  default, inspect the log pipeline for a `resource` processor that uses `action: upsert` on
  `deployment.environment.name`. Change the default to `action: insert`, after `k8s_attributes`, so that the workload
  annotation or resource attribute wins.
- **Exception:** in pods whose containers have different services, with per-container labels such as
  `tags.datadoghq.com/<container name>.service`, don't set the pod-level `resource.opentelemetry.io/service.name` and
  `resource.opentelemetry.io/service.version` annotations: they'd apply the same values to all the containers. Set the
  values in the SDK variables of each container. The `k8s_attributes` processor doesn't overwrite the attributes that
  the SDK's telemetry already has.
- **Fix:**
  - Keep the labels while the Datadog Agent collects telemetry about these workloads, and read the OpenTelemetry
    values from them with the downward API, or, in a shared Helm chart, with the same chart value, so that each value
    is set once. An annotation can't read a label: write the annotation with the same value. With the variables of
    [S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator):

    ```yaml
    spec:
      template:
        metadata:
          labels:
            tags.datadoghq.com/service: <service name>
            tags.datadoghq.com/version: <service version>
          annotations:
            resource.opentelemetry.io/service.name: <service name>
            resource.opentelemetry.io/service.version: <service version>
        spec:
          containers:
            - env:
                - name: OTEL_SERVICE_NAME
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.labels['tags.datadoghq.com/service']
                - name: OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.labels['tags.datadoghq.com/version']
    ```

  - Remove the `DD_SERVICE`, `DD_VERSION`, and `DD_ENV` variables that the manifests set, after copying their values
    to the OpenTelemetry variables and annotations, and to the matching `tags.datadoghq.com/*` labels when the
    workload doesn't have them: the Datadog Agent tags the telemetry it collects about the containers from the labels
    or these variables. Keep the variables when a remaining Datadog-specific feature requires them, as the [application
    workflow](#3-move-compatible-configuration-to-environment-variables) describes. Then check the injected ones: the
    Datadog Cluster Agent's Admission Controller doesn't mutate pods that have the
    `admission.datadoghq.com/enabled: "false"` label
    ([O-05](#o-05-annotate-workloads-and-prevent-double-instrumentation),
    [S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)), and injects them in the other
    pods.
  - When the workload has no Datadog `env` and the OpenTelemetry Collector sets `deployment.environment.name`, use the
    same value for a `tags.datadoghq.com/env` label that you add later
    ([C-11](#c-11-set-the-deployment-environment)).
- **Verify:** check the effective variables of a running pod with
  `kubectl exec <pod> --namespace <namespace> -- env | grep -E '^(DD|OTEL)_'`, then check in Datadog that the traces,
  logs, and container metrics of the workload have the same `service`, `version`, and `env` tags.

### S-12: Send OTLP to the Datadog Agent instead of standalone OpenTelemetry Collectors

- **Requirement:** when an OpenTelemetry Collector DaemonSet deployed with the OpenTelemetry community tools isn't an
  option ([C-14](#c-14-choose-where-the-opentelemetry-collector-runs-on-kubernetes)), the DDOT SDKs export OTLP to
  the Datadog Agent DaemonSet, in decreasing order of preference:
  - **The DDOT Collector bundled in the Datadog Agent:** enable `spec.features.otelCollector.enabled` in the
    `DatadogAgent`, which runs an OpenTelemetry Collector in the `otel-agent` container of the Datadog Agent, on ports
    4317 and 4318. It has the same support of the OpenTelemetry Collector components as the standalone DDOT Collector.
    When you override the Datadog Agent image tag, use 7.67.0 or later, without the `-full` suffix: the Datadog
    Operator rejects a `-full` tag for this feature ([DDOT Collector in the Datadog Agent][dd-ddot-collector-agent]).
  - **OTLP ingest in the Datadog Agent:** enable `spec.features.otlp.receiver.protocols.grpc.enabled` and
    `spec.features.otlp.receiver.protocols.http.enabled`, which open ports 4317 and 4318. For logs, also enable
    `spec.features.logCollection.enabled`, and set `DD_OTLP_CONFIG_LOGS_ENABLED=true` in
    `spec.override.nodeAgent.containers.agent.env`: OTLP logs ingestion is off by default, and requires Datadog Agent
    7.48.0 or later ([OTLP ingest in the Datadog Agent][dd-otlp-ingest]).
- **Requirement:** point the SDKs at the Datadog Agent of their own node: the Datadog Agent doesn't accept OTLP from
  another node. In the container `env`, replace the `daemon` OpenTelemetry Collector endpoint of
  [S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator):
  - **With the bundled DDOT Collector,** use the local Service of the Datadog Agent, `<DatadogAgent name>-agent`, the
    same convention as `DD_AGENT_HOST`. The Datadog Operator adds the ports 4317 and 4318 to this Service, with
    `internalTrafficPolicy: Local`, so that it reaches the Datadog Agent of the same node, on Kubernetes 1.22 or later:

    ```yaml
    - name: OTEL_EXPORTER_OTLP_ENDPOINT
      value: http://<DatadogAgent name>-agent.<namespace>.svc.cluster.local:4318
    ```

  - **With OTLP ingest, or on Kubernetes versions before 1.22,** use the node IP. The Datadog Operator publishes the
    ports of the bundled DDOT Collector as host ports by default, and the
    [OTLP ingest in the Datadog Agent][dd-otlp-ingest] docs use this endpoint:

    ```yaml
    - name: HOST_IP
      valueFrom:
        fieldRef:
          fieldPath: status.hostIP
    - name: OTEL_EXPORTER_OTLP_ENDPOINT
      value: http://$(HOST_IP):4318
    ```

- **Fix, with the bundled DDOT Collector: add the `k8s_attributes` processor.** Datadog supports it as a community
  component of the OpenTelemetry Collector Contrib, and it applies the OpenTelemetry Kubernetes conventions to all the
  telemetry ([C-14](#c-14-choose-where-the-opentelemetry-collector-runs-on-kubernetes)). The Datadog Operator doesn't
  set it up: its default configuration of the bundled DDOT Collector only has the `infraattributes` processor, and the
  ClusterRole of the Datadog Agent doesn't allow `k8s_attributes` to watch the pods. Three changes are needed:
  1. **The Collector configuration:** in `spec.features.otelCollector.conf.configData`, which replaces the default
     configuration, add the `k8s_attributes` processor of
     [C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs), with its `filter`,
     `extract.otel_annotations: true`, `extract.metadata`, and `pod_association`, to every pipeline that receives the
     telemetry of the workloads. Keep `infraattributes` after it, to add the Datadog Agent tagger's tags. Start from
     the default configuration, as the next item explains.
  2. **The node name:** `k8s_attributes` only watches the pods of its node with `filter.node_from_env_var`, but the
     Datadog Operator only gives the `otel-agent` container the node IP, in `DD_KUBERNETES_KUBELET_HOST`. Add the
     node name in the `DatadogAgent`:

     ```yaml
     spec:
       override:
         nodeAgent:
           containers:
             otel-agent:
               env:
                 - name: K8S_NODE_NAME
                   valueFrom:
                     fieldRef:
                       fieldPath: spec.nodeName
     ```

  3. **The RBAC:** grant the ServiceAccount of the Datadog Agent DaemonSet the permissions of the
     [`k8s_attributes` RBAC][otel-k8sattributes-rbac], the same ones that the Datadog Operator grants its OpenTelemetry
     Agent Gateway, plus `get`, `list`, and `watch` on `nodes` if you extract node metadata, such as `k8s.node.uid`
     or node labels ([C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs)). Find the
     ServiceAccount name with
     `kubectl get daemonset --namespace datadog -o jsonpath='{.items[*].spec.template.spec.serviceAccountName}'`:

     ```yaml
     apiVersion: rbac.authorization.k8s.io/v1
     kind: ClusterRole
     metadata:
       name: datadog-agent-k8s-attributes
     rules:
       - apiGroups: [""]
         resources: ["pods", "namespaces"]
         verbs: ["get", "watch", "list"]
       - apiGroups: ["apps"]
         resources: ["replicasets"]
         verbs: ["get", "watch", "list"]
     ---
     apiVersion: rbac.authorization.k8s.io/v1
     kind: ClusterRoleBinding
     metadata:
       name: datadog-agent-k8s-attributes
     roleRef:
       apiGroup: rbac.authorization.k8s.io
       kind: ClusterRole
       name: datadog-agent-k8s-attributes
     subjects:
       - kind: ServiceAccount
         name: <ServiceAccount of the Datadog Agent DaemonSet>
         namespace: datadog
     ```

- **The starting configuration of the bundled DDOT Collector.** `configData` replaces the
  [default configuration of the Datadog Operator][dd-operator-otel-default-config]: copy that default into
  `configData`, add `k8s_attributes` before `infraattributes` in its `traces`, `metrics`, and `logs` pipelines, and
  keep the rest. The default configuration has:
  - an `otlp` receiver on ports 4317 (gRPC) and 4318 (HTTP);
  - a `datadog` exporter and a `datadog/connector`, and a `traces` pipeline that exports to both;
  - the `infraattributes` processor, with `cardinality: 2`, in every pipeline;
  - a `prometheus` receiver that scrapes the internal metrics of the OpenTelemetry Collector on port 8888.

  The `otel-agent` container then completes this configuration with the DDOT converter
  ([converter README][ddot-converter]). The converter is a step of the startup of the `otel-agent` container: it reads
  the configuration, and adds the components that Datadog features need, when the configuration doesn't define them
  already. It only changes the pipelines that export to a `datadog` exporter, and names the components it adds with a
  `/dd-autoconfigured` suffix. By default, it adds:
  - the `infraattributes` processor, at the end of the pipelines that don't have one;
  - a `prometheus` receiver for the internal metrics, in its own pipeline, when no pipeline scrapes them. It sets the
    job name to `datadog-agent`, so that the health metrics of the OpenTelemetry Collector get the
    `service:datadog-agent` tag;
  - the `health_check`, `pprof`, `zpages`, and `ddflare` extensions;
  - from Datadog Agent 7.76.0, the `datadog` extension, which reports the OpenTelemetry Collector to
    [Fleet Automation][dd-fleet-automation];
  - from Datadog Agent 7.83.0, the `cumulativetodelta` processor, in the metrics pipelines.

  Keep the converter enabled, which is the default: without it, the bundled DDOT Collector loses the Datadog Agent
  tagger's tags, its health metrics, and its part of the Datadog Agent flare. With the default configuration and the
  converter, the OpenTelemetry Collector rules apply as follows:

  | Rule                                                            | With the bundled DDOT Collector                                                                                                                                                              |
  |-----------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
  | [C-01](#c-01-export-to-datadog-with-the-datadog-exporter)       | Already in the default configuration.                                                                                                                                                        |
  | [C-02](#c-02-compute-apm-trace-metrics-with-the-datadog-connector-before-any-sampling) | Already met while nothing samples the traces. The SDKs keep all the traces without the Datadog Agent's sampling rates (S-13): to sample, split the `traces` pipeline as C-02 does.                                                              |
  | [C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs) | Add `k8s_attributes`, with the three changes above.                                                                                                            |
  | [C-04](#c-04-dont-stamp-the-opentelemetry-collectors-own-identity-on-incoming-telemetry) | Only when you add a `resource_detection` processor.                                                                                                  |
  | [C-06](#c-06-convert-cumulative-metrics-to-delta)               | From Datadog Agent 7.83.0, the converter adds `cumulativetodelta`. Before 7.83.0, add it when the SDKs export cumulative metrics. The converter never adds `delta_to_rate`: add it as C-06 says. |
  | [C-07](#c-07-report-the-opentelemetry-collectors-to-fleet-automation-with-the-datadog-extension) | Nothing to add: from Datadog Agent 7.76.0, the converter adds the `datadog` extension, and reuses one that you define. Before 7.76.0, the `ddflare` extension reports to Fleet Automation. |
  | [C-08](#c-08-export-the-collectors-internal-metrics-through-its-own-pipeline) | Keep the default `prometheus` receiver instead of the OTLP loop of C-08: without a receiver that scrapes the internal metrics, the converter adds one, and the internal metrics reach Datadog twice. |
  | [C-09](#c-09-receive-otlp-over-both-grpc-and-http)              | Already in the default configuration.                                                                                                                                                        |
  | [C-10](#c-10-use-the-same-kubernetes-cluster-name-everywhere)   | Apply it when the telemetry needs `k8s.cluster.name`, with the cluster name of the Datadog Agent ([A-06](#a-06-use-the-same-kubernetes-cluster-name-as-the-opentelemetry-collectors)). |
  | [C-11](#c-11-set-the-deployment-environment)                    | Apply it with a `resource` processor when the cluster runs one environment.                                                                                                                  |
  | [C-12](#c-12-collect-middleware-telemetry-from-the-daemon-opentelemetry-collector-on-the-same-node) | Possible with `receiver_creator` and `k8s_observer`, which need more RBAC. Only when you need it.                                                     |
  | [C-13](#c-13-keep-the-daemon-opentelemetry-collectors-service-node-local) | Doesn't apply: the local Service of the Datadog Agent is already node-local.                                                                                       |

- **With the [`datadog` Helm chart][dd-helm-chart],** the bundled DDOT Collector differs from the Datadog Operator's
  (read from the chart's source, version 3.276.0, not tested):
  - **Endpoint:** the local Service is named after the chart's release, the `datadog.fullname` of the chart, or
    `agents.localService.overrideName`. The chart adds the ports of `datadog.otelCollector.ports`, 4317 and 4318 by
    default, to this Service, with `internalTrafficPolicy: Local`, on Kubernetes 1.22 or later. Port 8126 of the
    Datadog Agent is also on it, even with `datadog.apm.portEnabled: false`.
  - **Image:** the `otel-agent` container uses the `ddot-collector` image, with the tag of `agents.image.tag`, from
    Datadog Agent 7.67.0 (`datadog.otelCollector.useStandaloneImage: true`, the default). The chart rejects a tag with
    the `-full` suffix in this mode.
  - **Configuration:** `datadog.otelCollector.config` replaces the default configuration, as `configData` does.
  - **RBAC:** when `datadog.otelCollector.config` has a `k8s_attributes` processor, the chart creates the ClusterRole
    of its pods, namespaces, and ReplicaSets itself (`datadog.otelCollector.rbac.create: true`, the default), so you
    can skip the third change above. Add `nodes` with `datadog.otelCollector.rbac.rules` if you extract node metadata.
    The chart doesn't detect the processor in a ConfigMap of `datadog.otelCollector.configMap`: add the rules there.
  - **Node name:** the chart doesn't give the `otel-agent` container the node name either: add `K8S_NODE_NAME` with
    `agents.containers.otelAgent.env`, as in the second change above.
- **Without `k8s_attributes`: the limits of the `infraattributes` processor.** When you can't add `k8s_attributes`
  to the bundled DDOT Collector, its `infraattributes` processor is the only enrichment
  ([`infraattributes` processor][dd-infraattributes]):
  - It looks up the entity of the telemetry, such as its container or pod, in the Datadog Agent tagger, from
    resource attributes such as `container.id`, `k8s.pod.uid`, or `k8s.namespace.name`. Without any of them, it can't
    enrich the telemetry: check that the SDK sets them.
  - It adds the Datadog Kubernetes tags of the entity, such as `kube_namespace` and `pod_name`, at the configured
    `cardinality`. The default, `0`, only adds the low-cardinality tags: set `cardinality: 2`, as the
    [DDOT Collector in the Datadog Agent][dd-ddot-collector-agent] example does, to add the pod and container tags.
  - It maps Unified Service Tagging to `service.name`, `deployment.environment`, and `service.version`, only for the
    attributes that the telemetry doesn't already have: the values of the SDK win. The tagger reads the
    `tags.datadoghq.com/*` labels, the `DD_ENV`, `DD_SERVICE`, and `DD_VERSION` variables, and the
    `OTEL_SERVICE_NAME` and `OTEL_RESOURCE_ATTRIBUTES` variables, but only their literal values in the pod spec: it
    skips the values set with `valueFrom`, such as the downward API variables of S-08 and the `OTEL_RESOURCE_ATTRIBUTES`
    that the OpenTelemetry Operator injects.
  - It doesn't read the `resource.opentelemetry.io/*` annotations. It can't set `service.instance.id`,
    `service.namespace`, or custom resource attributes on the telemetry that the OpenTelemetry Collector collects
    itself, such as pod logs and metrics.

  To follow the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes] with these limits:
  - **Set the resource attributes in the SDK,** with the environment variables of
    [S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator). They compute the attributes as
    the conventions do, including `service.instance.id`, whatever the destination of the telemetry.
  - **Add the Unified Service Tagging labels to the pod template,** with the same values as the OpenTelemetry
    attributes ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)). The
    Datadog Agent reads them, instead of the annotations, for the telemetry it collects about the containers, such as
    logs and metrics:

    | OpenTelemetry resource attribute                                | Set in the SDK with                               | Unified Service Tagging label         | Datadog tag                       |
    |-----------------------------------------------------------------|---------------------------------------------------|---------------------------------------|-----------------------------------|
    | `service.name`                                                  | `OTEL_SERVICE_NAME`                               | `tags.datadoghq.com/service`          | `service`                         |
    | `service.version`                                               | `OTEL_RESOURCE_ATTRIBUTES`                        | `tags.datadoghq.com/version`          | `version`                         |
    | `deployment.environment.name`                                   | `OTEL_RESOURCE_ATTRIBUTES`                        | `tags.datadoghq.com/env`              | `env`                             |
    | `k8s.namespace.name`, `k8s.pod.name`, `k8s.container.name`, ... | `OTEL_RESOURCE_ATTRIBUTES`, with the downward API | None: `infraattributes` adds the tags | `kube_namespace`, `pod_name`, ... |

    In pods that have several containers, use the per-container labels, such as
    `tags.datadoghq.com/<container name>.service`.
  - **Keep the `resource.opentelemetry.io/*` annotations only for the components that read them,** such as the
    OpenTelemetry Operator, which computes the attributes it injects from them.
  - **For custom labels and annotations,** such as a team label, map them to tags in the Datadog Agent configuration
    ([Kubernetes tag extraction][dd-k8s-tags]).
- **With OTLP ingest:** the limits are at least those of the bundled DDOT Collector without `k8s_attributes`:
  - there's no OpenTelemetry Collector pipeline to configure, so you can't add `k8s_attributes`, and the
    OpenTelemetry Collector rules, C-01 to C-13, don't apply;
  - the `infraattributes` processor runs in the OTLP ingest pipelines for traces and metrics from Datadog Agent
    7.71.0, but not for logs;
  - no OpenTelemetry Collector pipeline can set `deployment.environment.name`. The Datadog Agent applies its default
    `env` to the traces that arrive without one. When the workloads need another value, set it in the SDKs
    ([C-11](#c-11-set-the-deployment-environment)), with the value of the `tags.datadoghq.com/env` label.
- **Datadog Agent features:** OTLP logs need `spec.features.logCollection.enabled`, which
  [A-01](#a-01-disable-the-features-that-duplicate-the-opentelemetry-collectors) disables in the reference
  configuration. Keep it enabled.
- **Conflicts:** the Datadog Agent's log collection tails the logs of the containers, so logs that the SDK also exports
  over OTLP reach Datadog twice ([S-10](#s-10-send-each-log-line-to-datadog-once)). For a workload that exports its
  logs over OTLP, exclude its container from the Datadog Agent's log collection with the
  `ad.datadoghq.com/<container name>.logs_exclude: "true"` pod annotation, from Datadog Agent 7.45, or with
  `DD_CONTAINER_EXCLUDE_LOGS` ([container discovery management][dd-container-discovery]).
- **Verify:** the traces, metrics, and logs of the workload arrive in Datadog with the Kubernetes tags of the pod, and
  each log line appears once. With `k8s_attributes`, also check that the logs of the `otel-agent` container don't
  report RBAC errors, such as `pods is forbidden`.

### S-13: Migrate workloads from the Datadog SDKs

- **Requirement:** when the workloads already run Datadog SDKs in their default mode, switch them to the OpenTelemetry
  mode of the DDOT SDKs one workload at a time, and check what changes for each workload:
  - **Unchanged:** the APM operation names, for example `servlet.request` in Java, and the service-level RED metrics
    based on them, such as `trace.servlet.request.hits`, stay the same, so the monitors and dashboards based on them
    keep working. The DDOT SDK exports the Datadog operation name in the `operation.name` attribute of the OTLP spans,
    which Datadog uses as the operation name. This requires the same service name: take
    `service.name` from the existing `DD_SERVICE` or `tags.datadoghq.com/service` value, or from the service name in
    effect today ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)).
    Otherwise, the migrated workload appears as a new APM service.
  - **Resource names and HTTP tags:** with the OpenTelemetry semantic conventions of
    [S-01](#s-01-enable-the-opentelemetry-mode-of-the-ddot-sdks), `DD_TRACE_OTEL_SEMANTICS_ENABLED=true`, some
    resource names and the HTTP tags change:

    The operation name and resource name come from different compatibility paths. The DDOT SDK preserves the Datadog
    operation name in the `operation.name` OTLP attribute. The resource name follows the OpenTelemetry HTTP semantic
    conventions. Server spans can use the framework route template, but HTTP client spans normally have no route
    template and therefore fall back to the method unless a path mapping is configured. This avoids using raw,
    high-cardinality, or sensitive URL paths. The change happens in the SDK before export, not in the Collector.

    | Span                                             | Resource name with the Datadog SDK in its default mode | Resource name with the DDOT SDK                                  |
    |--------------------------------------------------|--------------------------------------------------------|------------------------------------------------------------------|
    | HTTP server, with a route of the framework       | `GET /users/{id}`                                      | `GET /users/{id}`, unchanged                                     |
    | HTTP server, without a route                     | From the URL, for example `GET /users/?`, or `404`     | The method only, for example `GET`                               |
    | HTTP client                                      | From the URL, for example `GET /users/?`               | The method only, for example `GET`, unless a path mapping matches |
    | Non-standard HTTP method                         | The method                                             | `_OTHER`, with the method in `http.request.method_original`      |

    The HTTP tags follow the OpenTelemetry semantic conventions, for example for the URL and the method, and the
    spans with an error status get `error.type`. Before the switch, list the monitors, SLOs, and dashboards of the
    service that filter on `resource_name` or on the HTTP tags, then, for each of them:
    - **update it** to the new resource names and tags;
    - **or, for the HTTP client spans, keep a path in the resource name** with
      `DD_TRACE_HTTP_CLIENT_PATH_RESOURCE_NAME_MAPPING`, which maps the request paths to resource names, for example
      `/users/*:/users/?` gives `GET /users/?`
      ([Java tracer configuration][dd-java-config]).

    Until these are ready, keep the workload on the Datadog SDK in its default mode.
  - **Runtime metrics:** their names change to the OpenTelemetry names, for example `jvm.memory.used` for the JVM.
    Datadog maps them for the service page and the default runtime dashboards
    ([OpenTelemetry runtime metrics][dd-otel-runtime-metrics]). Check the custom monitors and dashboards that use the
    Datadog names ([S-07](#s-07-enable-runtime-metrics-where-supported)).
  - **Trace sampling and ingestion:** the sampling rates of the Datadog Agent, which target about 10 traces per
    second per Datadog Agent, no longer apply. The Datadog Agent returns them in its responses to the traces that the
    Datadog SDKs send to port 8126, and the SDKs don't send traces there in OTLP mode. Remote Configuration doesn't
    carry these rates: it carries the sample rates and sampling rules set in Datadog, which still apply in OTLP mode.
    Without `DD_TRACE_SAMPLE_RATE`, sampling rules, or Remote Configuration sampling, `dd-java-agent` keeps all the
    traces in OTLP mode, as the OpenTelemetry `parentbased_always_on` sampler does, and the ingested spans of a busy
    service can grow a lot (OTEL-3477). With sampling rules, it also keeps the traces that no rule matches.
    Before the switch, record the ingested spans of the service, for example with the
    `datadog.estimated_usage.apm.ingested_spans` metric, and compare them after the switch of the canary workload.
    Then sample in one of two places:
    - **In the OpenTelemetry Collector, after the `datadog` connector,** in the `traces/sampling` pipeline of
      [C-02](#c-02-compute-apm-trace-metrics-with-the-datadog-connector-before-any-sampling). The connector sees all
      the traces, so the APM trace metrics stay exact. This is the default recommendation, as for the workloads that
      the OpenTelemetry Operator instruments ([O-04](#o-04-point-the-sdks-at-the-daemon-opentelemetry-collector)).
      For example, a `probabilistic_sampler` processor that keeps 10% of the traces, in the values of the
      OpenTelemetry Kube Stack Helm chart. Helm replaces lists, so repeat the `receivers` and `exporters` of the
      pipeline:

      ```yaml
      collectors:
        daemon:
          config:
            processors:
              probabilistic_sampler:
                sampling_percentage: 10
            service:
              pipelines:
                traces/sampling:
                  receivers: [datadog/connector]
                  processors: [probabilistic_sampler]
                  exporters: [datadog/exporter]
      ```

    - **In the SDK,** with `DD_TRACE_SAMPLE_RATE`, `DD_TRACE_SAMPLING_RULES`, or Remote Configuration sampling, which
      still apply in OTLP mode, with the `DD_TRACE_RATE_LIMIT` limit, 100 traces per second by default. This also
      reduces the network traffic. The SDK only exports the sampled traces, with their sampling probability in the
      `th` value of the W3C `tracestate`, from `dd-java-agent` 1.67.0. The `datadog` connector scales the APM trace
      metrics back up with this probability from Datadog Agent 7.82.0, in the bundled DDOT Collector and the DDOT
      Collector: with earlier versions, the trace metrics only count the sampled traces.

    Keep the existing `DD_TRACE_SAMPLE_RATE`, `DD_TRACE_SAMPLING_RULES`, and `-Ddd.trace.sampl*` settings: they still
    apply ([S-09](#s-09-configure-a-java-agent-bundled-in-the-container-image)).
  - **Continuous Profiler:** it stops for the migrated workloads, because in-process profiling isn't available in the
    DDOT SDKs, such as the CPU, memory, and lock profiling of the Datadog SDKs. A workload uses it when
    `DD_PROFILING_ENABLED=true` or `-Ddd.profiling.enabled=true` is set or injected today
    ([S-06](#s-06-enable-optional-datadog-features-per-workload)). For these workloads:
    - **Switch to the DDOT Full Host Profiler,** the recommendation. It profiles the CPU usage of every process of the
      node with eBPF, and this guide's `install` script deploys it with `DD_HOST_PROFILER_ENABLED=true`
      ([host profiler][host-profiler]). The workload can then adopt the OpenTelemetry instrumentation, but with less
      visibility than the in-process profiler of the Datadog SDKs: there's no memory or lock profiling. Tell the user
      about this switch in the report, and don't carry `DD_PROFILING_ENABLED` into the configuration of the DDOT SDK.
    - **Only for the workloads that depend on memory or lock profiling,** keep them on the Datadog SDKs in their
      default mode.
  - **Trace and log correlation:** keep `DD_LOGS_INJECTION=true` in the DDOT SDKs, the same way as with the Datadog
    SDKs, so that the log lines carry the trace and span IDs, and check that the logs still link to the traces after
    the migration, whichever log path the workload uses ([S-10](#s-10-send-each-log-line-to-datadog-once)).
  - **Infrastructure telemetry,** when the OpenTelemetry Collectors replace the Datadog Agent's collection
    ([C-14](#c-14-choose-where-the-opentelemetry-collector-runs-on-kubernetes) level 1, with
    [A-01](#a-01-disable-the-features-that-duplicate-the-opentelemetry-collectors)): the Kubernetes metrics and the pod
    logs then come from the OpenTelemetry Collectors, with OpenTelemetry resource attributes instead of the tags that
    the Datadog Agent sets, such as `kube_namespace` or `source`. Check the monitors, dashboards, and log pipelines that
    filter on the Datadog Agent's tags before disabling its features.
- **Requirement:** every propagation configuration includes the W3C Trace Context (`tracecontext`) and W3C Baggage
  (`baggage`) styles, in the DDOT SDKs and in the Datadog SDKs that still run in their default mode:
  `DD_TRACE_PROPAGATION_STYLE`, `DD_TRACE_PROPAGATION_STYLE_EXTRACT`, `DD_TRACE_PROPAGATION_STYLE_INJECT`,
  `OTEL_PROPAGATORS`, and the `-Ddd.trace.propagation.*` system properties in Java. `tracecontext` keeps the traces
  connected with any OpenTelemetry SDK, and `baggage` carries the OpenTelemetry baggage between services. When these
  settings aren't set, the default list applies, which includes both. The `datadog` style is less useful now: the
  recent versions of all the Datadog SDKs support and enable `tracecontext` and `baggage` by default
  ([trace context propagation][dd-trace-context-propagation]). Keep `datadog` only for the workloads that exchange
  traces with older Datadog SDK versions, or with services configured with the `datadog` style alone that you can't
  change.
- **Requirement:** keep the traces connected between the migrated workloads and the ones that still run Datadog SDKs
  in their default mode. The W3C Trace Context format, `tracecontext`, connects them: the recent versions of all the
  Datadog SDKs extract and inject the `datadog`, `tracecontext`, and `baggage` propagation styles by default
  ([trace context propagation][dd-trace-context-propagation]), and the migrated workloads keep `tracecontext`:
  - The DDOT SDK Java agent has the same default list. `OTEL_PROPAGATORS` replaces it: with `tracecontext,baggage`,
    which [O-04](#o-04-point-the-sdks-at-the-daemon-opentelemetry-collector) sets through the `Instrumentation`'s
    `spec.propagators`, a migrated workload stops sending and reading the Datadog headers, but still sends and reads
    `tracecontext`.
  - **Check the workloads that still run Datadog SDKs:** the traces stay connected when they use the default
    propagation styles, or any list that includes `tracecontext`. Look for `DD_TRACE_PROPAGATION_STYLE`,
    `DD_TRACE_PROPAGATION_STYLE_EXTRACT`, and `DD_TRACE_PROPAGATION_STYLE_INJECT`, and their `-Ddd.trace.propagation.*`
    system properties in Java, set without `tracecontext` or `baggage`, for example to `datadog`, and add the missing
    styles. Also check the versions of their Datadog SDKs: for example, the Datadog SDK for .NET only injects
    `tracecontext` from version 2.22.0.
  - **Only when some of them don't extract or inject `tracecontext`,** and you can't change them first, keep the
    `datadog` style in the migrated workloads during the migration. The `Instrumentation`'s `spec.propagators` doesn't
    accept `datadog`, so set `DD_TRACE_PROPAGATION_STYLE` in its `spec.env`. It takes precedence over
    `OTEL_PROPAGATORS`, which the Datadog SDKs ignore when a `DD_TRACE_PROPAGATION_STYLE*` variable is set:

    ```yaml
    spec:
      env:
        - name: DD_TRACE_PROPAGATION_STYLE
          value: datadog,tracecontext,baggage
    ```

    Without the OpenTelemetry Operator, don't set `OTEL_PROPAGATORS`, so that the default list applies. Once these
    workloads are migrated or extract `tracecontext`, remove `DD_TRACE_PROPAGATION_STYLE`.
- **Requirement: move the pod logs one workload at a time,** when the `daemon` OpenTelemetry Collector replaces the log
  collection of the Datadog Agent ([C-14](#c-14-choose-where-the-opentelemetry-collector-runs-on-kubernetes) level
  1). At levels 2 and 3, the Datadog Agent keeps tailing the pod logs, and nothing moves. During the migration, keep
  the log collection of the Datadog Agent enabled for the whole cluster, with `spec.features.logCollection.enabled`,
  and move the pod logs of each workload to the `daemon` OpenTelemetry Collector when you migrate it:
  - Add the `ad.datadoghq.com/logs_exclude: "true"` annotation to the pod template of the workload, for example with
    the `openTelemetry.podLogs.enabled` value of a shared Helm chart, so that the Datadog
    Agent stops collecting its logs, from Datadog Agent 7.45 ([container discovery management][dd-container-discovery]).
  - Make the `file_log` receiver of the `daemon` OpenTelemetry Collector tail the logs of the workload. Its default
    `include`, `/var/log/pods/*/*/*.log`, tails every pod: during the migration, limit it to the migrated workloads,
    for example with one pattern per migrated namespace or workload, of the form
    `/var/log/pods/<namespace>_<pod name prefix>*_*/*/*.log`, so that the logs of the other workloads aren't collected
    twice.
  - Disable the log collection of the Datadog Agent
    ([A-01](#a-01-disable-the-features-that-duplicate-the-opentelemetry-collectors)) only at the end of the migration,
    once the logs of all the workloads come from the `daemon` OpenTelemetry Collector. Then restore the default
    `include` of `file_log`, and remove the `ad.datadoghq.com/logs_exclude` annotations.
- **Fix: roll out one workload at a time.** The OpenTelemetry mode is a configuration of the same Datadog SDK, so
  switching a workload doesn't need a new container image. Make the one-time image changes of
  [S-09](#s-09-configure-a-java-agent-bundled-in-the-container-image) first, such as the startup script or the
  `dd-java-agent` version, and keep the workloads in the default mode while they roll out. Then switch a
  workload by adding the variables of [S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)
  to its container `env`, for example with the `openTelemetry.apmSdk.enabled` value of a shared Helm chart, or by
  annotating its pods for the
  OpenTelemetry Operator ([O-05](#o-05-annotate-workloads-and-prevent-double-instrumentation)). Keep the Datadog Agent
  and its APM port, 8126, during the migration, for the workloads that still run Datadog SDKs in their default mode.
- **Rollback:** remove the variables of the OpenTelemetry mode
  ([S-01](#s-01-enable-the-opentelemetry-mode-of-the-ddot-sdks)) and the OTLP endpoint from the container `env`, or
  remove the OpenTelemetry Operator's annotation, then restart the workload: the Datadog SDK sends its telemetry to the
  Datadog Agent again. Also remove the `admission.datadoghq.com/enabled: "false"` label
  ([S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)), so that the Datadog Cluster
  Agent's Admission Controller injects its `DD_*` variables again, and the `ad.datadoghq.com/logs_exclude` annotation
  and the workload's `file_log` pattern, so that the Datadog Agent collects its logs again. With a shared Helm chart,
  turn `openTelemetry.apmSdk.enabled` and `openTelemetry.podLogs.enabled` off for the workload instead
  ([S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)).

  With GitOps, for example Argo CD or Flux, set the two values to `false` in the workload's values in Git, and let the
  pipeline deploy them: a rollback then takes one pipeline run. Don't change them directly in the cluster, with
  `helm` or `kubectl`: with self-heal on, Argo CD reverts direct changes to what Git says, and the workload switches
  back to the DDOT SDK.

  When Helm deploys the release directly, set the two values explicitly to `false` in the complete desired values for
  the release. Don't rely on omitting a canary values file: the previous value can remain in the Helm release or be
  reintroduced by another values layer.

  ```shell
  helm upgrade <release> <chart> \
    --reuse-values \
    --set <workload-values-path>.openTelemetry.apmSdk.enabled=false \
    --set <workload-values-path>.openTelemetry.podLogs.enabled=false
  ```

  Use `--reset-values` only when the command also supplies every required release override: resetting the release can
  remove unrelated configuration.

  Check the rendered and running workload:
  - the OpenTelemetry-mode variables and OTLP endpoint are absent;
  - the `admission.datadoghq.com/enabled: "false"` label is absent;
  - the `ad.datadoghq.com/logs_exclude: "true"` annotation is absent;
  - the Datadog Admission Controller variables and volumes are injected again;
  - the startup configuration reports that OTLP signals are disabled;
  - a marked request reaches the Datadog Agent and remains connected to downstream services.
- **Verify:** for each migrated workload, compare its APM service before and after the migration: the same `service`
  and `env` tags as the values recorded before the switch
  ([S-11](#s-11-align-datadog-unified-service-tagging-with-the-opentelemetry-service-identity)), a `version` tag, the
  same operation names, and connected traces with the workloads it calls and the ones that call it.

FIXME document the enrichments that the Datadog Agent adds to the pod logs it collects, and that the logs tailed by the
`daemon` OpenTelemetry Collector don't get, such as the `source` tag, for example `source:java`, which selects the
Datadog log integration pipelines, and how to restore them.

FIXME document how to parse the pod logs tailed by the `daemon` OpenTelemetry Collector to extract the trace and span
IDs that Datadog log injection adds, such as `dd.trace_id` and `dd.span_id`, into the trace context of the log records,
and which log layouts to use, so that the logs stay linked to the traces.

## Datadog Agent and Datadog Operator rules

These rules use the Datadog Operator's `DatadogAgent` custom resource (`datadoghq.com/v2alpha1`), as in
[`datadog-agent.yaml`][dd-agent-yaml]. With the [`datadog` Helm chart][dd-helm-chart], apply the same changes with
the chart values of this table. The table is read from the chart's source, version 3.276.0, and isn't tested in this
repository (OTEL-3478): report the changes made with the chart as `untested`.

| `DatadogAgent` of the Datadog Operator                         | Values of the `datadog` Helm chart                                                          |
|----------------------------------------------------------------|---------------------------------------------------------------------------------------------|
| `spec.global.clusterName`                                      | `datadog.clusterName`                                                                       |
| `spec.global.tags`                                             | `datadog.tags`                                                                              |
| `spec.override.nodeAgent.env`                                  | `datadog.env`, for all the containers of the Datadog Agent DaemonSet                        |
| `spec.override.nodeAgent.containers.agent.env`                 | `agents.containers.agent.env`                                                               |
| `spec.override.nodeAgent.containers.otel-agent.env`            | `agents.containers.otelAgent.env`                                                           |
| `spec.override.clusterAgent.env`                               | `clusterAgent.env`                                                                          |
| `spec.features.otelCollector.enabled`                          | `datadog.otelCollector.enabled`                                                             |
| `spec.features.otelCollector.conf.configData`                  | `datadog.otelCollector.config`, or an existing ConfigMap with `datadog.otelCollector.configMap` |
| `spec.features.otlp.receiver.protocols.grpc.enabled`, `.http.enabled` | `datadog.otlp.receiver.protocols.grpc.enabled`, `.http.enabled`                      |
| `DD_OTLP_CONFIG_LOGS_ENABLED=true`                             | `datadog.otlp.logs.enabled: true`                                                           |
| `spec.features.logCollection.enabled`                          | `datadog.logs.enabled`, with `datadog.logs.containerCollectAll`                             |
| `spec.features.kubeStateMetricsCore.enabled`                   | `datadog.kubeStateMetricsCore.enabled`                                                      |
| `spec.features.orchestratorExplorer.enabled`                   | `datadog.orchestratorExplorer.enabled`                                                      |
| `spec.features.eventCollection.collectKubernetesEvents`        | `datadog.collectEvents`                                                                     |
| `spec.features.clusterChecks.enabled`                          | `datadog.clusterChecks.enabled`                                                             |
| `spec.features.liveProcessCollection.enabled`                  | `datadog.processAgent.processCollection`                                                    |
| `spec.features.processDiscovery.enabled`                       | `datadog.processAgent.processDiscovery`                                                     |
| `spec.features.dynamicInstrumentation.enabled`                 | `datadog.dynamicInstrumentationGo.enabled`                                                  |
| `spec.features.apm.instrumentation.enabled`                    | `datadog.apm.instrumentation.enabled`                                                       |
| The Datadog Cluster Agent's Admission Controller `mutateUnlabelled` setting | `clusterAgent.admissionController.mutateUnlabelled`                            |

The chart also differs on the bundled DDOT Collector
([S-12](#s-12-send-otlp-to-the-datadog-agent-instead-of-standalone-opentelemetry-collectors)), and its
`datadog.profiling.enabled` value makes the Datadog Cluster Agent's Admission Controller inject `DD_PROFILING_ENABLED`
in the pods ([S-06](#s-06-enable-optional-datadog-features-per-workload)).

### A-01: Disable the features that duplicate the OpenTelemetry Collectors

- **Requirement:** disable a Datadog Agent feature only when its equivalent is enabled in the OpenTelemetry Collectors,
  to avoid collecting the same telemetry twice. The reference [`values.yaml`][values] enables all these equivalents
  except Kubernetes events, but the user may have changed the installer's defaults or configured OpenTelemetry Collectors without
  some of them: check each equivalent in the user's OpenTelemetry Collector configuration before disabling the Datadog Agent feature,
  and keep the Datadog Agent feature when its equivalent isn't enabled. The presets add their receivers to the
  pipelines: check the receivers in the rendered `OpenTelemetryCollector` custom resources, not only in the values.

  | Datadog Agent feature                                                                  | OpenTelemetry Collector equivalent, in the `opentelemetry-kube-stack` chart                                                                                                                                          |
  |----------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
  | `kubeStateMetricsCore`, and the legacy `kubernetes_state` check (`DD_IGNORE_AUTOCONF`) | `cluster` OpenTelemetry Collector scraping kube-state-metrics (`prometheus/ksm` receiver), and its `clusterMetrics` preset (`k8s_cluster` receiver)                                                                                |
  | `orchestratorExplorer`, `DD_ORCHESTRATOR_EXPLORER_KUBELET_CONFIG_CHECK_ENABLED`        | `cluster` OpenTelemetry Collector's `kubernetesObjects` preset (`k8s_objects` receiver), and `orchestrator_explorer.enabled: true` on its Datadog exporter                                                                         |
  | `eventCollection.collectKubernetesEvents`                                              | `kubernetesEvents` preset (`k8s_objects` receiver watching the `events.k8s.io` events), on one OpenTelemetry Collector only. The `kubernetesObjects` preset doesn't collect events. **Not enabled in the reference `values.yaml`**       |
  | `logCollection`                                                                        | `daemon` OpenTelemetry Collector's `logsCollection` preset (`file_log` receiver), tailing the container logs, enabled by the chart's `daemon` defaults                                                                             |
  | `clusterChecks`                                                                        | OpenTelemetry Collector receivers for the cluster-level services, such as the middleware of [C-12](#c-12-collect-middleware-telemetry-from-the-daemon-opentelemetry-collector-on-the-same-node)                                                  |
  | `liveProcessCollection`, `processDiscovery`, `DD_DISCOVERY_ENABLED`                    | No exact equivalent: disabled in the reference to keep the Datadog Agent to its enabler role. Keep them if the user uses them                                                                                        |

  The reference configuration disables the Datadog Agent's Kubernetes events collection although the OpenTelemetry Collectors don't
  collect them. To collect Kubernetes events, enable the `kubernetesEvents` preset on the `cluster` OpenTelemetry Collector, or keep
  `eventCollection.collectKubernetesEvents` enabled in the Datadog Agent:

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/datadog-agent.yaml path=.spec.features snippet-path=.spec.features mode=subset sha=da0c09aa1264 -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/datadog-agent.yaml path=.spec.override.nodeAgent.containers.agent.env snippet-path=.spec.override.nodeAgent.containers.agent.env mode=subset sha=9c17ce9913bc -->
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
  keep it, and say so in the report. If the user prefers to keep a Datadog Agent feature, disable its equivalent in
  the OpenTelemetry Collectors instead, rather than collecting the telemetry twice. To keep all of them, for example
  to migrate only the SDKs first, see the values of the "Level 1 for APM only" conflict of
  [C-14](#c-14-choose-where-the-opentelemetry-collector-runs-on-kubernetes). If `DD_IGNORE_AUTOCONF` is already set,
  append `kubernetes_state` to its space-separated list.
  During a migration from the Datadog SDKs, keep `logCollection` enabled until the end, and exclude the migrated
  workloads pod by pod ([S-13](#s-13-migrate-workloads-from-the-datadog-sdks)).

### A-02: Keep the Datadog Cluster Agent

- **Requirement:** don't disable the Datadog Cluster Agent: the Datadog Agent running on each node relies on it for
  cluster-level metadata, and the Datadog Operator doesn't fully support running without it
  ([datadog-operator#3521][dd-operator-3521]).

### A-03: Enable Live Debugger and Dynamic Instrumentation

- **Requirement:**

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/datadog-agent.yaml path=.spec.features.dynamicInstrumentation snippet-path=.spec.features.dynamicInstrumentation mode=exact sha=26b3426b2593 -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/datadog-agent.yaml path=.spec.override.nodeAgent.containers.agent.env[name=DD_APM_DEBUGGER_LOGS_ENABLED_OVERRIDE] snippet-path=.spec.override.nodeAgent.containers.agent.env[name=DD_APM_DEBUGGER_LOGS_ENABLED_OVERRIDE] mode=exact sha=a5a8c3ce88de -->
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
  that the `daemon` OpenTelemetry Collector, which tails the container logs, parses the Datadog Agent's logs. See the
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
- **Pod logs tailed by the `daemon` OpenTelemetry Collector:** at level 1 of
  [C-14](#c-14-choose-where-the-opentelemetry-collector-runs-on-kubernetes), when the Collector tails the pod logs
  instead of the Datadog Agent, the logs miss enrichments of the Datadog Agent, such as the `source` tag that selects
  the Datadog log integration pipelines, and this guide doesn't yet document how to parse the `dd.trace_id` and
  `dd.span_id` that log injection adds, to keep the logs linked to the traces (FIXMEs of
  [S-13](#s-13-migrate-workloads-from-the-datadog-sdks)). Keep the Datadog Agent tailing the pod logs of the
  workloads that depend on these, and report the move of their logs as `blocked`.
- **Live Debugger for Go on arm64 with Linux >= 6.18:** Go probes fail to load with
  `invalid bpf_context access off=328 size=8` ([datadog-agent#57314][dd-agent-57314]). Use x86_64 nodes or an older
  arm64 kernel.

## Verify and report

Build and run each resulting container with the same command and runtime that production uses. Generate identifiable
traffic and test telemetry. Use a unique validation resource attribute, and emit a known span, metric point, and log
record for each enabled signal.

Verify all of the following:

1. **Activation:** process arguments, startup logs, or SDK diagnostics prove that one intended loader, profiler,
   runtime agent such as a Java agent, or compiled integration is active.
2. **Effective configuration:** inspect the running container's environment and SDK diagnostics. Confirm service
   identity, whether application code or the injected SDK registers the `TracerProvider`, `MeterProvider`, and
   `LoggerProvider` and configures their exporters, enabled signals, destinations, and shared versus signal-specific
   precedence. Redact headers and credentials from retained evidence.
3. **OpenTelemetry Collector receipt:** OpenTelemetry Collector telemetry and temporary debug output show the unique test data for every enabled
   signal. A healthy receiver without the test record is insufficient.
4. **Datadog receipt:** find the same test data in Trace Explorer, Metrics Explorer, and Log Explorer as applicable.
   Check `service`, `env`, `version`, and expected semantic-convention attributes.
5. **Runtime metrics:** when enabled, find runtime-specific metric names in Datadog and record whether they traveled by
   OTLP or DogStatsD. Successful application metrics do not establish runtime metrics delivery.
6. **No duplicates:** confirm one expected trace tree per request, one copy of the marked log record, and the expected
   metric series and points. Check both OpenTelemetry Collector and Datadog Agent paths.
7. **Idempotence:** run the coding agent against the adapted repository again. It must report the setup as satisfied
   without changing dependencies, activation, global `TracerProvider`, `MeterProvider`, or `LoggerProvider`
   registration, exporter configuration, or deployment configuration.

For Kubernetes deployments, also verify:

1. **Rendered configuration:** run `helm template` with the deployment values, or
   `kubectl apply --dry-run=server`. Compare the result with [`examples/default/rendered/`][examples-rendered].
2. **Runtime resources:** confirm that every OpenTelemetry Collector, OpenTelemetry Operator, Datadog Operator, and Datadog Agent
   component present in the selected architecture is healthy. A deployment without a Datadog Agent and manual
   instrumentation without an Operator are valid. Confirm that each instrumented pod has the intended DDOT SDK and
   effective `OTEL_*` and required `DD_*` environment.

   ```sh
   kubectl get pods --namespace <collectors-namespace>
   kubectl get pods --namespace datadog
   kubectl get pod <instrumented-pod> --namespace <namespace> -o yaml | grep -E 'image:|OTEL_|DD_'
   ```

3. **Platform data in Datadog:** for components present in the architecture, confirm that [Fleet
   Automation][dd-fleet-automation] lists the Datadog Agents and OpenTelemetry Collectors under the Kubernetes cluster name and that
   **OpenTelemetry Collector** dashboard shows no refused or failed telemetry. Confirm that the services appear in APM
   with their environment and Kubernetes tags.

Record a result for every attempted application:

| Field | Required evidence |
|---|---|
| Source | Application revision, guide revision, fixture revision, and starting state |
| Versions | Runtime, operating system, architecture, SDK package or image digest, OpenTelemetry Collector, and Datadog Agent |
| Activation | Exact build and startup mechanism and evidence that it loaded |
| Configuration | Effective non-secret destination, protocol, enabled signals, resource identity, and whether application code or an injected SDK registers the `TracerProvider`, `MeterProvider`, and `LoggerProvider` and configures their exporters |
| Traces | OpenTelemetry Collector evidence, Datadog evidence, semantic attributes, and duplicate check |
| Application metrics | OpenTelemetry Collector evidence, Datadog evidence, temporality, and duplicate check |
| Application logs | OpenTelemetry Collector evidence, Datadog evidence, correlation, and duplicate check |
| Runtime metrics | Enabled state, transport, metric names, Datadog evidence, or explicit reason it was not enabled |
| Conflicts | Deliberate disables, secondary exporters, `TracerProvider`, `MeterProvider`, or `LoggerProvider` registration in application code, activation mechanisms, and preserved decisions |
| Outcome | `passed`, `failed`, `blocked`, or `untested`, with the reason |

A blocked or untested signal is not a pass. Limit the final claim to the recorded language, versions, runtime,
operating system, architecture, deployment type, and signals. See [TROUBLESHOOTING.md][troubleshooting] when telemetry
does not appear.

[install]: ./INSTALL.md
[host-profiler]: ./README.md#host-profiler-optional
[values]: ./values.yaml
[dd-agent-yaml]: ./datadog-agent.yaml
[cert-manager]: https://cert-manager.io/
[dd-helm-chart]: https://github.com/DataDog/helm-charts/tree/main/charts/datadog
[examples-rendered]: ./examples/default/rendered/
[troubleshooting]: ./TROUBLESHOOTING.md
[architecture]: ./assets/deployment-architecture-diagram_k8s_ddot-with-dd-agent.svg.png
[cluster-name]: https://github.com/DataDog/opentelemetry-examples/blob/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/README.md#kubernetes-cluster-name-constraints
[dd-fleet-automation]: https://docs.datadoghq.com/agent/fleet_automation/
[dd-otel-extension-resource]: https://docs.datadoghq.com/opentelemetry/integrations/datadog_extension/
[dd-live-debugger-go]: https://docs.datadoghq.com/tracing/live_debugger/?prog_lang=go
[dd-ddi]: https://docs.datadoghq.com/tracing/trace_collection/dynamic_instrumentation/
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
[otel-semconv-v1-22-0]: https://github.com/open-telemetry/semantic-conventions/releases/tag/v1.22.0
[otel-k8sattributes-semconv]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/processor/k8sattributesprocessor/README.md#semantic-conventions-compatibility
[otel-helm-charts-2538]: https://github.com/open-telemetry/opentelemetry-helm-charts/issues/2538
[dd-hostname-mapping]: https://docs.datadoghq.com/opentelemetry/mapping/hostname/?tab=datadogexporter
[dd-collector-setup]: https://docs.datadoghq.com/opentelemetry/setup/collector_exporter/
[dd-otel-env]: https://docs.datadoghq.com/opentelemetry/config/environment_variable_support/
[dd-otel-api]: https://docs.datadoghq.com/opentelemetry/instrument/dd_sdks/api_support/
[dd-otlp-export]: https://docs.datadoghq.com/opentelemetry/instrument/dd_sdks/otlp_trace_export/
[dd-java-config]: https://docs.datadoghq.com/tracing/trace_collection/library_config/java/
[dd-otlp-ingest]: https://docs.datadoghq.com/opentelemetry/setup/otlp_ingest_in_the_agent/?tab=kubernetesoperator
[dd-ddot-collector-agent]: https://docs.datadoghq.com/opentelemetry/setup/ddot_collector/install/kubernetes_daemonset/
[dd-container-discovery]: https://docs.datadoghq.com/containers/guide/container-discovery-management/
[dd-k8s-tags]: https://docs.datadoghq.com/containers/kubernetes/tag/
[otel-collector-chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-collector
[otel-k8sattributes-rbac]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/k8sattributesprocessor#role-based-access-control
[dd-infraattributes]: https://github.com/DataDog/datadog-agent/blob/main/comp/otelcol/otlp/components/processor/infraattributesprocessor/README.md
[dd-operator-otel-default-config]: https://github.com/DataDog/datadog-operator/blob/main/internal/controller/datadogagent/feature/otelcollector/defaultconfig/defaultconfig.go
[ddot-converter]: https://github.com/DataDog/datadog-agent/blob/main/comp/otelcol/converter/README.md
[dd-trace-context-propagation]: https://docs.datadoghq.com/tracing/trace_collection/trace_context_propagation/
[dd-runtime-metrics]: https://docs.datadoghq.com/tracing/metrics/runtime_metrics/
[dd-otel-runtime-metrics]: https://docs.datadoghq.com/opentelemetry/integrations/runtime_metrics/
[dd-deployment-tracking]: https://docs.datadoghq.com/tracing/services/deployment_tracking/
[otel-go]: https://opentelemetry.io/docs/languages/go/
[otel-sdk-env-vars]: https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/
[otel-go-autoexport]: https://pkg.go.dev/go.opentelemetry.io/contrib/exporters/autoexport
[otel-go-autoprop]: https://pkg.go.dev/go.opentelemetry.io/contrib/propagators/autoprop
[k8s-dependent-env-vars]: https://kubernetes.io/docs/tasks/inject-data-application/define-interdependent-environment-variables/
[dd-agent-network]: https://docs.datadoghq.com/agent/configuration/network/#open-ports
[dd-trace-go-5407]: https://github.com/DataDog/dd-trace-go/pull/5407
[dotnet-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/dotnet-core/
[go-setup]: https://docs.datadoghq.com/tracing/trace_collection/automatic_instrumentation/dd_libraries/go/
[java-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/java/?tab=wget
[node-setup]: https://docs.datadoghq.com/tracing/trace_collection/automatic_instrumentation/dd_libraries/nodejs/
[python-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/python/
