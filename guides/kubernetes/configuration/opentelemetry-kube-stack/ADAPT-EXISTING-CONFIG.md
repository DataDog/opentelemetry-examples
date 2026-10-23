# Adapt an existing configuration for DDOT

Already running an OpenTelemetry Collector, the OpenTelemetry Operator, OpenTelemetry SDKs, or a Datadog Agent on
Kubernetes? This guide lists what to change in each of them so they work together with the DDOT SDKs, Datadog's version
of the OpenTelemetry APM SDKs, without reinstalling from scratch with the [`install` script][install].

Each requirement is a rule with an ID, what to check, how to fix it, and how it can conflict with an existing setup. The
reference implementation of every rule is in this folder: [`values.yaml`][values] for the OpenTelemetry Kube Stack Helm
chart, and [`datadog-agent.yaml`][dd-agent-yaml] for the Datadog Operator's `DatadogAgent` custom resource. When this
guide and those files disagree, the files are right.

> [!TIP]
> **Using a coding agent?** Give it this prompt from the repository that holds your Kubernetes configuration:
>
> ```text
> Adapt the OpenTelemetry and Datadog configuration of this repository for DDOT, following
> https://raw.githubusercontent.com/DataDog/opentelemetry-examples/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/ADAPT-EXISTING-CONFIG.md
> ```

Contents:

- [Target architecture](#target-architecture)
- [Instructions for coding agents](#instructions-for-coding-agents)
- [OpenTelemetry Collector rules](#opentelemetry-collector-rules)
- [OpenTelemetry Operator and `Instrumentation` rules](#opentelemetry-operator-and-instrumentation-rules)
- [DDOT SDK environment rules](#ddot-sdk-environment-rules)
- [Datadog Agent and Datadog Operator rules](#datadog-agent-and-datadog-operator-rules)
- [Known gaps](#known-gaps)
- [Verify](#verify)

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

## Instructions for coding agents

1. **Find the components** in the repository: OpenTelemetry Collector configurations (Helm values of the
   `opentelemetry-kube-stack` or `opentelemetry-collector` charts, `OpenTelemetryCollector` custom resources, raw
   Collector YAML), `Instrumentation` custom resources, workload manifests with `instrumentation.opentelemetry.io/*`
   annotations or `OTEL_*`/`DD_*` environment variables, `DatadogAgent` custom resources, and Helm values of the
   `datadog` or `datadog-operator` charts. Only apply the rule sections of the components you found.
2. **Merge, don't replace.** Keep the user's existing receivers, exporters, pipelines, and environment variables unless
   a rule says otherwise. When a rule conflicts with an existing setting, follow the rule's **Conflicts** line, and if
   it doesn't cover the case, ask the user.
3. **Helm replaces lists.** When writing a Helm overlay, repeat every item of a list you override (pipeline
   `receivers`/`processors`/`exporters`, `env`, `operators`), or the items of the base values file are lost.
4. **Don't invent values.** Use placeholders such as `<CLUSTER_NAME>` for the Kubernetes cluster name, namespaces, and
   Service names you can't find in the repository, and list them in your report. Never write API or application keys
   into files: reference a Kubernetes Secret.
5. **Pin image versions** to the ones in [`values.yaml`][values] and [`datadog-agent.yaml`][dd-agent-yaml] at the time
   you read them, not to the versions quoted in this guide, which can be older.
6. **Report** a table of every rule: `applied`, `already satisfied`, `skipped` (with the reason), or `blocked` (by a
   [known gap](#known-gaps) or a missing value), followed by the placeholders the user must fill in.

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

### C-03: Add Kubernetes metadata to application telemetry

- **Requirement:** the `k8s_attributes` processor in every pipeline that receives OTLP from the applications.
- **Fix:** `presets.kubernetesAttributes.enabled: true` with the kube-stack chart, or the `k8s_attributes` processor
  and its RBAC otherwise.

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

### C-05: Set `host.name` from the Kubernetes node name

- **Requirement:** on the `daemon` collector, the `logs`, `traces`, and `metrics` pipelines set `host.name` from
  `k8s.node.name` when it's missing, so that application telemetry is attached to the same host as the Datadog Agent and
  the infrastructure metrics.
- **Fix:**

  ```yaml
  processors:
    resource/hostname:
      attributes:
        - action: insert
          key: host.name
          from_attribute: k8s.node.name
  ```

### C-06: Convert cumulative metrics to delta

- **Requirement:** the `cumulativetodelta` processor in the metrics pipelines receiving cumulative metrics (OTLP,
  Prometheus receivers), and `deltatorate` for the counters that Datadog displays as rates.
- **Reference:** `defaultCRConfig.config.processors` and `collectors.daemon.config.processors` in [`values.yaml`][values].
- **Conflicts:** applications should also export delta metrics, see [S-04](#s-04-export-delta-metrics).
- **Conflicts:** when a metrics pipeline also exports to destinations that require cumulative temporality, such as
  Prometheus remote write or another observability backend, don't convert to delta in that shared pipeline: split it
  with a `forward` connector into a fan-out with two ends. The cumulative end exports without conversion, and the
  Datadog end adds `cumulativetodelta` and `deltatorate` before the Datadog exporter. Keep the shared processors
  (`k8s_attributes`, `resource_detection`, `resource/hostname`) in the upstream pipeline, and the `datadog/connector`
  as a receiver of the Datadog end only:

  ```yaml
  connectors:
    forward/metrics: {}
  service:
    pipelines:
      metrics:                         # shared: receive and enrich once
        receivers: [otlp]
        processors: [k8s_attributes, resource_detection/env, resource/hostname]
        exporters: [forward/metrics]
      metrics/cumulative:              # destinations requiring cumulative temporality
        receivers: [forward/metrics]
        exporters: [prometheusremotewrite/other-backend]
      metrics/datadog:                 # Datadog requires delta temporality
        receivers: [forward/metrics, datadog/connector]
        processors: [cumulativetodelta, deltatorate]
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
  `clusterName` ([A-06](#a-06-use-the-same-kubernetes-cluster-name-as-the-collectors)).
- **Fix:** on EKS, GKE, and AKS, enable the cloud provider's resource detection preset, which discovers it.
- **Fix:** on other platforms, where `k8s.cluster.name` is set manually instead of discovered, pass it to the
  Collectors through the `OTEL_RESOURCE_ATTRIBUTES` environment variable, in addition to setting it on the telemetry
  with a `resource` processor: the Collector's Datadog extension
  ([C-07](#c-07-report-the-collectors-to-fleet-automation-with-the-datadog-extension)) reads it from
  `OTEL_RESOURCE_ATTRIBUTES`, not from the pipelines' processors, to report the Collector under its Kubernetes cluster
  in Fleet Automation. With the kube-stack chart, `resourceAttributes.k8s.cluster.name` in the values does both:

  ```yaml
  resourceAttributes:
    k8s.cluster.name: <CLUSTER_NAME>
  ```

  Otherwise, append `k8s.cluster.name=<CLUSTER_NAME>` to the Collector container's `OTEL_RESOURCE_ATTRIBUTES`, keeping
  its existing comma-separated attributes, and add a `resource` processor that upserts `k8s.cluster.name` to the
  pipelines.
- **Constraints:** dot-separated tokens of lowercase letters, digits, and `-`, starting with a letter and ending with a
  letter or a digit, at most 80 characters. See [Kubernetes cluster name constraints][cluster-name].

### C-11: Set the deployment environment

- **Requirement:** `deployment.environment.name` on all the telemetry, for example with `resourceAttributes` in the
  kube-stack chart's values.
- **Workaround:** also set the deprecated `deployment.environment` to the same value until
  [dd-trace-java#12324][dd-trace-java-12324] ships.
- FIXME change the description of the resolution of the workaround to point to the DDOT Go SDK as it's the last one to release.  

### C-12: Collect middleware telemetry from the `daemon` collector on the same node

- **Requirement:** collect the telemetry of the middleware running in the Kubernetes cluster, such as database
  servers, message brokers, or caches, with the `daemon` collector running on the same Kubernetes node as the observed
  pod. Not with the `cluster` collector, a standalone Collector Deployment, or a `daemon` collector on another node.
- **Why:** the Collector enriches the telemetry it collects with the resource attributes of the observed pod, and
  `host.name` comes from that pod's `k8s.node.name`
  ([C-05](#c-05-set-host-name-from-the-kubernetes-node-name)). Collected from another node, or through a Kubernetes
  Service that load balances across pods, the telemetry gets the wrong resource attributes, for example the
  Collector's `host.name` or another replica's `k8s.pod.name`, and is attached to the wrong host in Datadog.
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
        processors: [k8s_attributes, resource_detection/env, resource/hostname, cumulativetodelta]
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
  - the points of a time series can reach different Collector instances, which breaks the stateful `cumulativetodelta`
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
        value: /otel-auto-instrumentation-dotnet/linux/Datadog.Tracer.Native.so
      - name: DD_DOTNET_TRACER_HOME
        value: /otel-auto-instrumentation-dotnet
  ```

- FIXME Link to a github issue on the DDOT SDK .Net repository, id dd-trace-dotnet#issue-number, that tracks the fix.
- 
### O-03: Raise the Python auto-instrumentation volume size

- **Requirement:** `python.volumeLimitSize: 500Mi`.
- **Why:** the DDOT SDK for Python image bundles native builds for every supported Python version, libc, and CPU
  architecture, about 330 MB, more than the OpenTelemetry Operator's 200 Mi default: the pod would be evicted.
- **Conflicts:** the OpenTelemetry Operator warns that the field is deprecated in favor of `volume.size`, which isn't
  in its CRD schema yet: keep `volumeLimitSize`.
- FIXME indicate when the fix will be rolled out on the DDOT Python SDK image, so the user can revert to the default 200 Mi and this rule can be removed.

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
- **Workaround:** set `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` to `<daemon collector endpoint>/v1/traces` in the
  `Instrumentation`'s `env`, as the DDOT SDK for Go ignores `OTEL_EXPORTER_OTLP_ENDPOINT`
  ([dd-trace-go#5407][dd-trace-go-5407]).
- See also the [known gaps](#known-gaps) for Go.

## DDOT SDK environment rules

Set these variables in the `Instrumentation` custom resource's `spec.env`, so they apply to every instrumented workload.
A variable set in a workload container's `env` takes precedence for that workload. When applications are configured
without the OpenTelemetry Operator, set them in their environment instead.

### S-01: Enable the OpenTelemetry mode of the DDOT SDKs

- **Requirement:**

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

### S-02: Point the DDOT SDKs at the Datadog Agent

- **Requirement:** `DD_AGENT_HOST` is the DNS name of the Datadog Agent's Service, for example
  `datadog-agent.datadog.svc.cluster.local` for a `DatadogAgent` named `datadog` in the `datadog` namespace.
- **Why:** the DDOT SDKs export their telemetry over OTLP to the OpenTelemetry Collector, but rely on the Datadog Agent for their
  Datadog-specific features, such as Remote Configuration and Live Debugger.

### S-03: Export over OTLP/HTTP

- **Requirement:** force the OTLP/HTTP protocol, whatever the SDK's default:

  ```yaml
  - name: OTEL_EXPORTER_OTLP_PROTOCOL
    value: http/protobuf
  ```

- **Why:** the SDKs' default protocol differs per language, for example gRPC for Node.js. Forcing OTLP/HTTP makes
  every SDK export to the same `daemon` collector port, 4318, the one of the `Instrumentation`'s endpoint
  ([O-04](#o-04-point-the-sdks-at-the-daemon-collector)).
- **Conflicts:** if the user's endpoint is the gRPC port 4317, change it to 4318 together with the protocol.
- FIXME some DDOT SDKs don't support gRPC yet, so the user must use OTLP/HTTP for now. Indicate which ones, and when they will support gRPC.

### S-04: Export delta metrics

- **Requirement:** use the delta temporality, the one Datadog requires:

  ```yaml
  - name: OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE
    value: delta
  ```

- **Why:** the SDKs then export delta metrics, which the Collector forwards to Datadog without converting them, so
  without the memory and CPU cost of the conversion.
- **Conflicts:** when the user requires cumulative temporality, for example because the OpenTelemetry Collector also
  exports the metrics to destinations that require it, set
  `OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE=cumulative`, and convert all the metrics from cumulative to delta
  in the downstream part of the Collector pipeline that exports to Datadog
  ([C-06](#c-06-convert-cumulative-metrics-to-delta)). Then make sure the Collectors have the resources for that
  conversion:
  - `cumulativetodelta` keeps the last value of every time series in memory: size the Collectors' memory limit for
    the number of time series of the workloads they receive from, and watch their memory usage on the
    **OpenTelemetry Collector** dashboard after the change. The reference [`values.yaml`][values] allocates `1Gi` of
    memory and `500m` of CPU per Collector: scale up for workloads with many time series.
  - Set `max_staleness` on `cumulativetodelta`, for example `max_staleness: 1h`, so that the time series that stop
    reporting, such as the ones of deleted pods, are evicted from memory.
  - All the points of a time series must reach the same Collector instance for the conversion to be correct: keep the
    conversion in the `daemon` collector receiving the workload's telemetry, behind a node-local Service
    ([C-13](#c-13-keep-the-daemon-collectors-service-node-local)), not in a Collector Deployment with several replicas
    behind a load-balanced Service.
- FIXME call out that DDOT SDKs only support delta temporality for now, so the user must use delta metrics for now and the cumulative to delta processor in the collector

### S-05: Use the stable semantic conventions

- **Requirement:** `OTEL_SEMCONV_STABILITY_OPT_IN=http,database,rpc,messaging`.

### S-06: Enable optional Datadog features per workload

These features are disabled by default. Enable them in the `Instrumentation`'s `env` for every workload, or in a
workload's container `env` for that workload only:

| Feature                                    | Variables                                                                              |
|--------------------------------------------|----------------------------------------------------------------------------------------|
| [Database Monitoring][dd-dbm]              | `DD_DBM_PROPAGATION_MODE=full`, `DD_DBM_TRACE_PREPARED_STATEMENTS=true`                |
| [Data Streams Monitoring][dd-dsm]          | `DD_DATA_STREAMS_ENABLED=true`, `DD_TRACE_REMOVE_INTEGRATION_SERVICE_NAMES_ENABLED=true` |
| [App & API Protection][dd-aap]             | `DD_APPSEC_ENABLED=true`                                                               |
| [Code Security IAST][dd-iast]              | `DD_IAST_ENABLED=true`                                                                 |
| [Code Security SCA][dd-sca]                | `DD_APPSEC_SCA_ENABLED=true`                                                           |

- **Conflicts:** if the user's workloads already set any of these, keep their values.
- **Precedence:** the same environment variable can be set in several places. Before changing one, find all the places
  it's set for the workload, to know which value applies. From the highest to the lowest precedence:
  1. the `env` of the container in the pod manifest: the OpenTelemetry Operator doesn't inject a variable the
     container's `env` already defines;
  2. the language section of the `Instrumentation` custom resource, for example `spec.java.env`, only applied with the
     `inject-<language>` annotations: the `inject-sdk` annotation, used for Go, ignores it;
  3. the `spec.env` of the `Instrumentation` custom resource;
  4. the container's `envFrom` (ConfigMaps, Secrets): the OpenTelemetry Operator only checks the container's `env`,
     so it injects the variable even when `envFrom` defines it, and the injected value wins;
  5. the `ENV` instructions of the image's `Dockerfile`, overridden by any of the above.

  For example, `DD_DATA_STREAMS_ENABLED=true` in the `Instrumentation`'s `spec.env` has no effect on a workload whose
  container `env` sets `DD_DATA_STREAMS_ENABLED=false`, but overrides `ENV DD_DATA_STREAMS_ENABLED=false` in its
  `Dockerfile`. The OpenTelemetry Operator injects the variables when the pod is created: restart the workloads after
  changing the `Instrumentation`. Check the effective values on a running pod:

  ```sh
  kubectl exec <pod> --namespace <namespace> -- env | grep -E '^(DD|OTEL)_'
  ```

  The pod spec shows the variables set in the manifest and injected by the OpenTelemetry Operator, but not those of
  the `Dockerfile` or `envFrom`: `kubectl exec ... env` is the only way to see the value the application gets.

  This order applies to the `DD_*` variables of this table and to the `OTEL_*` variables of the `Instrumentation`. The
  OpenTelemetry Operator handles a few variables differently: it appends its agent to `JAVA_TOOL_OPTIONS`,
  `NODE_OPTIONS`, and `PYTHONPATH`, and merges its resource attributes into `OTEL_RESOURCE_ATTRIBUTES`, instead of
  skipping them when the container's `env` sets them. It skips the injection of a language's SDK altogether when the
  container sets `JAVA_TOOL_OPTIONS`, `NODE_OPTIONS`, or `PYTHONPATH` with `valueFrom`. See `injectCommonEnvVar` and
  `appendIfNotSet` in the OpenTelemetry Operator's [`internal/instrumentation/sdk.go`][otel-operator-sdk-go].

## Datadog Agent and Datadog Operator rules

These rules use the Datadog Operator's `DatadogAgent` custom resource (`datadoghq.com/v2alpha1`), as in
[`datadog-agent.yaml`][dd-agent-yaml]. With the `datadog` Helm chart, apply the same changes with the chart's equivalent
values: this guide doesn't list them, and they aren't tested in this repository.

### A-01: Disable the features that duplicate the OpenTelemetry Collectors

- **Requirement:** disable a Datadog Agent feature only when its equivalent is enabled in the OpenTelemetry Collectors,
  to avoid collecting the same telemetry twice. The reference [`values.yaml`][values] enables all these equivalents
  except the Kubernetes events one, but the user may have changed the installer's defaults, or configured their own
  Collectors without some of them: check each equivalent in the user's Collector configuration before disabling the
  Agent feature, and keep the Agent feature when its equivalent isn't enabled. The presets add their receivers to the
  pipelines: check the receivers in the rendered `OpenTelemetryCollector` custom resources, not only in the values.

  | Datadog Agent feature                                                                  | OpenTelemetry Collector equivalent, in the `opentelemetry-kube-stack` chart                                                                                                                                          |
  |----------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
  | `kubeStateMetricsCore`, and the legacy `kubernetes_state` check (`DD_IGNORE_AUTOCONF`) | `cluster` collector scraping kube-state-metrics (`prometheus/ksm` receiver), and its `clusterMetrics` preset (`k8s_cluster` receiver)                                                                                |
  | `orchestratorExplorer`, `DD_ORCHESTRATOR_EXPLORER_KUBELET_CONFIG_CHECK_ENABLED`        | `cluster` collector's `kubernetesObjects` preset (`k8s_objects` receiver), and `orchestrator_explorer.enabled: true` on its Datadog exporter                                                                         |
  | `eventCollection.collectKubernetesEvents`                                              | `kubernetesEvents` preset (`k8s_objects` receiver watching the `events.k8s.io` events), on one collector only. The `kubernetesObjects` preset doesn't collect events. **Not enabled in the reference `values.yaml`** |
  | `logCollection`                                                                        | `daemon` collector's `logsCollection` preset (`file_log` receiver), tailing the container logs, enabled by the chart's `daemon` defaults                                                                             |
  | `clusterChecks`                                                                        | Collector receivers for the cluster-level services, such as the middleware of [C-12](#c-12-collect-middleware-telemetry-from-the-daemon-collector-on-the-same-node)                                                  |
  | `liveProcessCollection`, `processDiscovery`, `DD_DISCOVERY_ENABLED`                    | No exact equivalent: disabled in the reference to keep the Agent to its enabler role. Keep them if the user uses them                                                                                                |

  The reference configuration, which disables the Agent's Kubernetes events collection although the Collectors don't
  collect them: to collect the Kubernetes events, enable the `kubernetesEvents` preset on the `cluster` collector, or
  keep `eventCollection.collectKubernetesEvents` enabled in the Agent:

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
  keep it, and say so in the report. If the user prefers to keep an Agent feature, disable its equivalent in the
  Collectors instead, rather than collecting the telemetry twice. If `DD_IGNORE_AUTOCONF` is already set, append `kubernetes_state` to its space-separated
  list.
- FIXME misalignment to check: with the reference configuration, no component collects the Kubernetes events.
  `datadog-agent.yaml` disables the Agent's `eventCollection.collectKubernetesEvents`, but the Collectors don't
  collect them either: `values.yaml` disables the `kubernetesEvents` preset on the `daemon` collector and doesn't enable
  it on the `cluster` collector, and the `kubernetesObjects` preset doesn't collect events (verified against
  `opentelemetry-kube-stack` chart 0.24.2 and `examples/default/rendered/collector.yaml`). Decide which component
  collects them: enable `collectors.cluster.presets.kubernetesEvents` in `values.yaml` and re-render the examples, or
  re-enable `eventCollection.collectKubernetesEvents` in `datadog-agent.yaml`. Then simplify the events row of the table
  and the sentence introducing the reference configuration.

### A-02: Keep the Cluster Agent

- **Requirement:** don't disable the Datadog Cluster Agent: the node Agent relies on it for cluster-level metadata, and
  the Datadog Operator doesn't fully support running without it ([datadog-operator#3521][dd-operator-3521]).

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
  Agent detects it, unless [Fleet Automation][fleet-automation] Remote Configuration is enabled, which requires it.
- **Recommended on non-cloud Kubernetes clusters:** use `<node name>-<cluster name>` as the Agent hostname, unique
  across Kubernetes clusters:

  ```yaml
  spec:
    override:
      nodeAgent:
        env:
          # K8S_NODE_NAME and K8S_CLUSTER_NAME must be defined before DD_HOSTNAME.
          - name: K8S_NODE_NAME
            valueFrom:
              fieldRef: { fieldPath: spec.nodeName }
          - name: K8S_CLUSTER_NAME
            value: "<CLUSTER_NAME>"
          - name: DD_HOSTNAME
            value: "$(K8S_NODE_NAME)-$(K8S_CLUSTER_NAME)"
  ```
 - FIXME: verify this is needed

## Known gaps

Report these as `blocked` when the user's setup depends on them:

- **App & API Protection over OTLP:** with the DDOT SDK for Go exporting over OTLP, App & API Protection data isn't
  sent to Datadog (OTEL-3411).
- **OpenTelemetry API in Go compiled with Orchestrion:** spans created with the OpenTelemetry API aren't bridged to the
  DDOT SDK for Go ([orchestrion#936][orchestrion-936]).
- **Live Debugger for Go on arm64 with Linux >= 6.18:** Go probes fail to load with
  `invalid bpf_context access off=328 size=8` ([datadog-agent#57314][dd-agent-57314]). Use x86_64 nodes or an older
  arm64 kernel.

## Verify

1. **Render before applying.** Run `helm template` with the user's values, or `kubectl apply --dry-run=server`, and check
   that each rule appears in the rendered manifests. Compare with the rendered reference manifests in
   [`examples/default/rendered/`][examples-rendered].
2. **On the Kubernetes cluster:** the Collector, OpenTelemetry Operator, Datadog Operator, and Datadog Agent pods are
   running, and an instrumented pod has the DDOT SDK init container and the `OTEL_*` and `DD_*` environment variables:

   ```sh
   kubectl get pods --namespace <collectors-namespace>
   kubectl get pods --namespace datadog
   kubectl get pod <instrumented-pod> --namespace <namespace> -o yaml | grep -E 'image:|OTEL_|DD_'
   ```

3. **In Datadog:** [Fleet Automation][dd-fleet-automation] lists the Datadog Agents and the Collectors under the
   Kubernetes cluster name, the **OpenTelemetry Collector** dashboard shows no refused or failed telemetry, and the
   instrumented services show up in APM with their `env` and Kubernetes tags.

See [TROUBLESHOOTING.md][troubleshooting] when something doesn't show up.

[install]: https://github.com/DataDog/opentelemetry-examples/blob/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/INSTALL.md
[values]: https://raw.githubusercontent.com/DataDog/opentelemetry-examples/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/values.yaml
[dd-agent-yaml]: https://raw.githubusercontent.com/DataDog/opentelemetry-examples/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/datadog-agent.yaml
[examples-rendered]: https://github.com/DataDog/opentelemetry-examples/tree/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/examples/default/rendered
[troubleshooting]: https://github.com/DataDog/opentelemetry-examples/blob/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/TROUBLESHOOTING.md
[architecture]: https://raw.githubusercontent.com/DataDog/opentelemetry-examples/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/assets/deployment-architecture_otel-kube-stack-ddot.svg.png
[cluster-name]: https://github.com/DataDog/opentelemetry-examples/blob/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/README.md#kubernetes-cluster-name-constraints
[fleet-automation]: https://github.com/DataDog/opentelemetry-examples/blob/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/README.md#fleet-automation-optional
[dd-fleet-automation]: https://docs.datadoghq.com/agent/fleet_automation/
[dd-live-debugger-go]: https://docs.datadoghq.com/tracing/live_debugger/?prog_lang=go
[dd-dbm]: https://docs.datadoghq.com/database_monitoring/
[dd-dsm]: https://docs.datadoghq.com/data_streams/
[dd-aap]: https://docs.datadoghq.com/security/application_security/
[dd-iast]: https://docs.datadoghq.com/security/code_security/iast/
[dd-sca]: https://docs.datadoghq.com/security/code_security/software_composition_analysis/
[orchestrion]: https://github.com/DataDog/orchestrion
[orchestrion-936]: https://github.com/DataDog/orchestrion/issues/936
[dd-trace-go-5407]: https://github.com/DataDog/dd-trace-go/pull/5407  
[dd-trace-java-12324]: https://github.com/DataDog/dd-trace-java/pull/12324
[dd-operator-3521]: https://github.com/DataDog/datadog-operator/issues/3521
[dd-agent-57314]: https://github.com/DataDog/datadog-agent/issues/57314
[otel-operator-sdk-go]: https://github.com/open-telemetry/opentelemetry-operator/blob/v0.159.0/internal/instrumentation/sdk.go
