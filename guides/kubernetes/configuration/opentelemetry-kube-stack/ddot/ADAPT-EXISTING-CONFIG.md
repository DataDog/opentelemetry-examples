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
> https://raw.githubusercontent.com/DataDog/opentelemetry-examples/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/ADAPT-EXISTING-CONFIG.md
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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.env snippet-path=.env mode=subset -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.exporters["datadog/exporter"] snippet-path=.config.exporters["datadog/exporter"] mode=subset -->
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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.collectors.daemon.config.connectors snippet-path=.connectors mode=subset -->
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

### C-03: Enrich the telemetry with the resource attributes Datadog needs

- **Requirement:** the traces, metrics, and logs of a workload carry the same values of the following resource
  attributes: Datadog correlates the telemetry on them, and can't link the signals of a workload when they differ.
  `service.namespace` and `deployment.environment.name` are strongly recommended rather than required.

  | Resource attribute                                                                                                                                                                                                                  | Datadog use                                                                                           | Set by, in the reference configuration                                                                                                                                   |
  |-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
  | `service.name`                                                                                                                                                                                                                      | `service` tag, needed for any correlation                                                             | OpenTelemetry Operator and `k8s_attributes`, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]: annotations, labels, workload names      |
  | `service.namespace`                                                                                                                                                                                                                 | Groups the services of an application. Strongly recommended, not required                             | OpenTelemetry Operator and `k8s_attributes`, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes]: annotations, Kubernetes namespace        |
  | `deployment.environment.name`                                                                                                                                                                                                       | `env` tag. Strongly recommended                                                                       | OpenTelemetry Operator and `resourceAttributes` of the kube-stack chart ([C-11](#c-11-set-the-deployment-environment)): annotations, OTel Collector pipeline             |
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

    <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.processors.k8s_attributes.extract.metadata snippet-path=.processors.k8s_attributes.extract.metadata mode=subset -->
    <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.processors.k8s_attributes mode=adapted sha=9ba22a88e1c4 -->
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
  - **`datadog.host.name`:** don't set it where`k8s.cluster.name`, `k8s.node.name`, `host.id`, and the `cloud.*`
    attributes can be reliably discovered, such as on EKS, AKS, and GKE: Datadog then derives the hostname from them.
    Only set `datadog.host.name` when those can't be reliably discovered, for example on non-cloud Kubernetes clusters, set
    `datadog.host.name` to `<node name>-<cluster name>` in the `daemon` collector, the same value as the Datadog
    Agent's hostname ([A-06](#a-06-use-the-same-kubernetes-cluster-name-as-the-opentelemetry-collectors)), so that the 
    Agent and the Collectors report the same host:

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
  For those applications, also set it in the container's `OTEL_RESOURCE_ATTRIBUTES`
  ([S-08](#s-08-configure-the-sdks-of-workloads-without-the-opentelemetry-operator)).
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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.processors["resource_detection/env"] snippet-path=.processors["resource_detection/env"] mode=subset -->
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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.extensions.datadog.api snippet-path=.extensions.datadog.api mode=exact -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.collectors.daemon.config.extensions.datadog.deployment_type snippet-path=.extensions.datadog.deployment_type mode=exact -->
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.service.extensions snippet-path=.service.extensions mode=subset -->
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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.defaultCRConfig.config.service.telemetry snippet-path=.service.telemetry mode=subset -->
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

- **Recommendation:** strongly recommended, set `deployment.environment.name` on all the telemetry, for example with
  `resourceAttributes` in the kube-stack chart's values. Datadog maps it to the `env` tag of
  [Unified Service Tagging][dd-unified-service-tagging].

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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.dotnet.env snippet-path=.dotnet.env mode=exact -->
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

- FIXME Link to a github issue on the DDOT SDK .Net repository, id dd-trace-dotnet#issue-number, that tracks the fix.
- 
### O-03: Size the Python auto-instrumentation volume

- **Requirement:** use the DDOT SDK for Python image `4.16.1-preview` or later, which fits the OpenTelemetry Operator's
  200 Mi default volume. With `4.16.0` or earlier, set `python.volumeLimitSize: 500Mi`.
- **Why:** up to `4.16.0`, the DDOT SDK for Python image exceeds 200 Mi once copied into the auto-instrumentation
  volume: the pod would be evicted.
- **Conflicts:** the OpenTelemetry Operator warns that `volumeLimitSize` is deprecated in favor of `volume.size`, which
  isn't in its CRD schema yet: keep `volumeLimitSize` if you need it.
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

- **Recommendation:** instrument Go applications at build time with [Orchestrion][orchestrion] for the DDOT SDK for
  Go. Compile-time instrumentation is recommended and optional: see the alternative below.
- **Requirement:** when the OpenTelemetry Operator configures Go applications, use the
  `instrumentation.opentelemetry.io/inject-sdk` annotation, which only injects the `OTEL_*` configuration.
- **Requirement:** use the DDOT SDK for Go `v2.12.0` or later, which falls back to `OTEL_EXPORTER_OTLP_ENDPOINT` for
  traces ([dd-trace-go#5407][dd-trace-go-5407]).
- **Workaround:** with earlier versions, which ignore `OTEL_EXPORTER_OTLP_ENDPOINT`, set
  `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` to `<daemon collector endpoint>/v1/traces` in the `Instrumentation`'s `env`.
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

## DDOT SDK environment rules

Set these variables in the `Instrumentation` custom resource's `spec.env`, so they apply to every instrumented workload.
A variable set in a workload container's `env` takes precedence for that workload. When applications are configured
without the OpenTelemetry Operator, set them in their environment instead.

### S-01: Enable the OpenTelemetry mode of the DDOT SDKs

- **Requirement:** configure all of the following for workloads adopting DDOT SDKs with OpenTelemetry semantic
  conventions and OTLP export, whether starting from Datadog-native instrumentation or an existing OpenTelemetry
  setup. Verify the effective configuration at runtime; the variables' presence in `spec.env` is insufficient:

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.env mode=subset -->
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

- **Why:** together, these settings configure OpenTelemetry semantic conventions and OTLP export for the supported
  signals:
  - `DD_TRACE_OTEL_ENABLED=true` enables OpenTelemetry tracing support, which can also be used with Datadog-native
    export. This flag alone leaves traces exported with the Datadog protocol (the default). It does not switch to
    OTLP, so set `OTEL_TRACES_EXPORTER=otlp` explicitly. See
    [Export traces from Datadog SDKs in OTLP format][dd-otlp-trace-export] for supported versions and configuration,
    and check the SDK versions pinned in [`values.yaml`][values].
  - `DD_LOGS_OTEL_ENABLED=true` and `DD_METRICS_OTEL_ENABLED=true` enable the SDK's supported OTel logs and metrics
    pipelines with OTLP export. The configuration above enables both signals by default. If application code
    already creates its own OTel logs or metrics provider, check that provider's exporter separately: these flags
    may leave the existing setup unchanged.
  - `DD_TRACE_OTEL_SEMANTICS_ENABLED=true` switches instrumentation from Datadog semantic conventions to
    OpenTelemetry semantic conventions, including span names and attributes. It is also intended to enforce OTLP
    trace export. The reverse does not apply: `OTEL_TRACES_EXPORTER=otlp` does not by itself switch from Datadog
    to OpenTelemetry semantics. Keep both settings explicit, and verify that the pinned SDK version supports them
    and that workload-specific overrides do not disable either behavior.
  - `OTEL_TRACES_SPAN_METRICS_ENABLED=false` stops the SDK from computing span stats, so the Datadog connector of
    the Collector computes the APM trace metrics of every service
    ([C-02](#c-02-compute-apm-trace-metrics-with-the-datadog-connector-before-any-sampling)).
- **Fix:** merge the variables above into the `Instrumentation` configuration, preserving existing list entries.
  Before reporting `already satisfied` or `applied`, check that each workload uses this configuration and that its
  container, language-specific, or application settings do not override it. Keep `OTEL_TRACES_EXPORTER=otlp`
  explicitly set; `DD_TRACE_OTEL_ENABLED=true` alone does not satisfy this rule.
  For logs and metrics, verify that the pinned SDK's required OTel packages are available and that each enabled
  signal's pipeline exports to the intended collector before reporting the rule as satisfied.
- **Conflicts:** preserve explicitly disabled signals (`DD_*_OTEL_ENABLED=false`, `OTEL_SDK_DISABLED=true`, or
  `OTEL_*_EXPORTER=none`) and alternative exporters until the user confirms the intended migration.
  Report the conflict and ask whether to keep the existing setting or migrate to DDOT with OTLP export.

### S-02: Point the DDOT SDKs at the Datadog Agent

- **Requirement:** `DD_AGENT_HOST` is the DNS name of the Datadog Agent's Service, for example
  `datadog-agent.datadog.svc.cluster.local` for a `DatadogAgent` named `datadog` in the `datadog` namespace.
- **Why:** the DDOT SDKs export their telemetry over OTLP to the OpenTelemetry Collector, but rely on the Datadog Agent for their
  Datadog-specific features, such as Remote Configuration and Live Debugger.

### S-03: Export over OTLP/HTTP

- **Requirement:** use OTLP/HTTP on port 4318. Keep the shared protocol from [`values.yaml`][values]:

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.env mode=subset -->
  ```yaml
  - name: OTEL_EXPORTER_OTLP_PROTOCOL
    value: http/protobuf
  ```

- **Why:** these settings direct the SDKs to the `daemon` collector's OTLP/HTTP endpoint on port 4318
  ([O-04](#o-04-point-the-sdks-at-the-daemon-collector)). Check which HTTP encodings each pinned SDK version
  supports. If a signal does not support the shared `http/protobuf` setting, configure a supported HTTP encoding
  for that signal without changing the others.
- **Conflicts:** check signal-specific protocol and endpoint overrides as well as the shared settings. If the user's
  endpoint is the gRPC port 4317, change it to 4318 together with the protocol. Use the
  [OTLP trace export guide][dd-otlp-trace-export] and the installed SDK version to check supported protocols.

### S-04: Export delta metrics

- **Requirement:** use the delta temporality, the one Datadog requires:

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.env mode=subset -->
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
  - `cumulative_to_delta` keeps the last value of every time series in memory: size the Collectors' memory limit for
    the number of time series of the workloads they receive from, and watch their memory usage on the
    **OpenTelemetry Collector** dashboard after the change. The reference [`values.yaml`][values] allocates `1Gi` of
    memory and `500m` of CPU per Collector: scale up for workloads with many time series.
  - Set `max_staleness` on `cumulative_to_delta`, for example `max_staleness: 1h`, so that the time series that stop
    reporting, such as the ones of deleted pods, are evicted from memory.
  - All the points of a time series must reach the same Collector instance for the conversion to be correct: keep the
    conversion in the `daemon` collector receiving the workload's telemetry, behind a node-local Service
    ([C-13](#c-13-keep-the-daemon-collectors-service-node-local)), not in a Collector Deployment with several replicas
    behind a load-balanced Service.
- FIXME call out that DDOT SDKs only support delta temporality for now, so the user must use delta metrics for now and the cumulative to delta processor in the collector

### S-05: Use the stable semantic conventions

- **Requirement:** `OTEL_SEMCONV_STABILITY_OPT_IN=http,database,rpc,messaging`.

### S-06: Enable optional Datadog features per workload

These features of the DDOT SDKs are disabled by default. Enable them in the `Instrumentation`'s `env` for every
workload, or in a workload's container `env` for that workload only. The commented-out entries of `instrumentation.env`
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

### S-07: Enable runtime metrics where supported

- **Recommendation:** enable runtime metrics for supported workloads to help troubleshoot memory usage, garbage
  collection, and thread activity. Configure this per workload; shared instrumentation defaults remain unchanged.
- **Fix:** set `DD_RUNTIME_METRICS_ENABLED=true` in the workload's container `env`. For OTLP export, keep
  `DD_METRICS_OTEL_ENABLED=true` and `OTEL_METRICS_EXPORTER=otlp` effective for that workload. Check the installed
  SDK version's runtime support, required packages or imports, and export transport before applying the change.
- **Compatibility:** check the [runtime metrics documentation][dd-runtime-metrics] for the installed SDK version,
  and verify its implementation when its DDOT behavior is not covered. If the SDK sends runtime metrics only
  through DogStatsD, they cannot reach the Collector through its OTLP endpoint. Keep the existing DogStatsD
  destination, such as the Datadog Agent. For an Agent-free setup, report runtime-metrics delivery as `blocked`
  until a supported delivery path is configured.
- **Verify:** confirm that runtime metrics from the workload arrive in Datadog with the expected service and
  environment tags through the intended transport. An enabled flag or successful trace export is insufficient.
- **Conflicts:** preserve explicit runtime-metrics disables and existing runtime instrumentation. Ask before
  enabling a disabled collector or adding another one that could duplicate the same measurements.

### S-08: Configure the SDKs of workloads without the OpenTelemetry Operator

- **Requirement:** when the OpenTelemetry Operator doesn't inject the OpenTelemetry configuration into a workload,
  add these variables to the `env` of the application container, replacing the `<...>` placeholders. This is
  typically the case when the OpenTelemetry SDKs are managed manually: the SDK is included in the container image of
  the service, and its OpenTelemetry configuration is managed in the Kubernetes deployment. The variables form the
  following groups:
  - **OpenTelemetry SDK configuration**, for all OpenTelemetry SDKs: the `daemon` collector endpoint
    ([O-04](#o-04-point-the-sdks-at-the-daemon-collector)), the OTLP/HTTP protocol
    ([S-03](#s-03-export-over-otlphttp)), the delta temporality ([S-04](#s-04-export-delta-metrics)), and the stable
    semantic conventions ([S-05](#s-05-use-the-stable-semantic-conventions)).
  - **Resource attributes**, for all OpenTelemetry SDKs, following the
    [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes].
  - **DDOT SDK configuration**, only for the DDOT SDKs: the OpenTelemetry mode
    ([S-01](#s-01-enable-the-opentelemetry-mode-of-the-ddot-sdks)), the Datadog Agent host
    ([S-02](#s-02-point-the-ddot-sdks-at-the-datadog-agent)), and `OTEL_TRACES_SPAN_METRICS_ENABLED=false`, which
    leaves the computation of the APM trace metrics to the Datadog connector of the Collector
    ([C-02](#c-02-compute-apm-trace-metrics-with-the-datadog-connector-before-any-sampling)). To also enable optional Datadog features, such as
    Dynamic Instrumentation, Database Monitoring, Data Streams Monitoring, App & API Protection, and Code Security,
    add their variables to this group ([S-06](#s-06-enable-optional-datadog-features-per-workload)).

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.env mode=adapted sha=d743bf605b4b -->
  ```yaml
  spec:
    containers:
      - env:
          ## OpenTelemetry SDK configuration
          - name: OTEL_EXPORTER_OTLP_ENDPOINT
            value: http://<helm-release-name>-daemon-collector.<namespace>.svc.cluster.local:4318
          - name: OTEL_EXPORTER_OTLP_PROTOCOL
            value: http/protobuf
          - name: OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE
            value: delta
          - name: OTEL_SEMCONV_STABILITY_OPT_IN
            value: http,database,rpc,messaging

          ## Resource attributes
          - name: OTEL_SERVICE_NAME
            value: <service name>
          - name: OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT_NAME
            value: <deployment environment name>
          - name: OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION
            value: <service version>
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
            value: deployment.environment.name=$(OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT_NAME),k8s.namespace.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAMESPACE),k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),k8s.pod.uid=$(OTEL_RESOURCE_ATTRIBUTES_POD_UID),k8s.container.name=$(OTEL_RESOURCE_ATTRIBUTES_CONTAINER_NAME),service.instance.id=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAMESPACE).$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).$(OTEL_RESOURCE_ATTRIBUTES_CONTAINER_NAME),service.namespace=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAMESPACE),service.version=$(OTEL_RESOURCE_ATTRIBUTES_SERVICE_VERSION)

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

- **Requirement:** in addition to the resource attribute variables above, which only configure the SDK, annotate
  the pod template with the `resource.opentelemetry.io/*` annotations, such as
  `resource.opentelemetry.io/service.name` and `resource.opentelemetry.io/service.namespace`
  ([C-03](#c-03-enrich-the-telemetry-with-the-resource-attributes-datadog-needs)). Use the same values as in the
  environment variables. The OpenTelemetry Collector produces telemetry about the same pod, such as its logs,
  metrics, and Kubernetes events, and can't read the SDK's environment variables. With the `k8s_attributes`
  processor and `extract.otel_annotations: true`, the Collector reads the annotations and enriches this telemetry
  with the same resource attributes, following the [OpenTelemetry Kubernetes conventions][otel-semconv-k8s-attributes].
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
  - **Organization of the telemetry:** `service.name`, `service.namespace`, `deployment.environment.name`, and the
    custom resource attributes that classify services let you group and filter the telemetry, for example by
    application and environment.
  - **APM troubleshooting:** some APM capabilities depend on specific attributes. For example, `service.version`
    enables [Deployment Tracking][dd-deployment-tracking], which compares the behavior of a service across its
    versions.
- **`deployment.environment.name`:** strongly recommended. When the Collector pipeline already sets it
  ([C-11](#c-11-set-the-deployment-environment)), you can remove `OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT_NAME`
  and the `deployment.environment.name` entry of `OTEL_RESOURCE_ATTRIBUTES`. The same manifests then serve all the
  environments.
- **Fix:** keep `OTEL_RESOURCE_ATTRIBUTES` after the variables it references. Kubernetes only expands a `$(VAR)`
  reference to a variable defined earlier in the same container's `env`, and leaves the other references as literal
  text ([dependent environment variables][k8s-dependent-env-vars]). For Go workloads that use a DDOT SDK for Go
  version earlier than `v2.12.0`, also set `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`
  ([O-06](#o-06-instrument-go-applications-at-compile-time)). With the upstream OpenTelemetry SDKs, leave out the
  DDOT SDK configuration group.
- **Conflicts:** if the container already sets `OTEL_RESOURCE_ATTRIBUTES`, merge its attributes into the new value
  and keep the user's values when they differ. For the other variables, follow the conflicts of the rules each group
  links to, such as [S-01](#s-01-enable-the-opentelemetry-mode-of-the-ddot-sdks) for explicitly disabled signals.

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

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/datadog-agent.yaml path=. mode=subset -->
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
  `opentelemetry-kube-stack` chart 0.26.1 and `examples/default/rendered/collector.yaml`). Decide which component
  collects them: enable `collectors.cluster.presets.kubernetesEvents` in `values.yaml` and re-render the examples, or
  re-enable `eventCollection.collectKubernetesEvents` in `datadog-agent.yaml`. Then simplify the events row of the table
  and the sentence introducing the reference configuration.

### A-02: Keep the Cluster Agent

- **Requirement:** don't disable the Datadog Cluster Agent: the node Agent relies on it for cluster-level metadata, and
  the Datadog Operator doesn't fully support running without it ([datadog-operator#3521][dd-operator-3521]).

### A-03: Enable Live Debugger and Dynamic Instrumentation

- **Requirement:**

  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/datadog-agent.yaml path=. mode=subset -->
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
  Agent detects it, unless [Fleet Automation][dd-fleet-automation] Remote Configuration is enabled, which requires it.
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

[install]: https://github.com/DataDog/opentelemetry-examples/blob/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/INSTALL.md
[values]: https://raw.githubusercontent.com/DataDog/opentelemetry-examples/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml
[dd-agent-yaml]: https://raw.githubusercontent.com/DataDog/opentelemetry-examples/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/datadog-agent.yaml
[examples-rendered]: https://github.com/DataDog/opentelemetry-examples/tree/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/examples/default/rendered
[troubleshooting]: https://github.com/DataDog/opentelemetry-examples/blob/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/TROUBLESHOOTING.md
[architecture]: https://raw.githubusercontent.com/DataDog/opentelemetry-examples/main/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/assets/deployment-architecture-diagram_k8s_ddot-with-dd-agent.svg.png
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
[dd-trace-go-5407]: https://github.com/DataDog/dd-trace-go/pull/5407  
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
[dd-otlp-trace-export]: https://docs.datadoghq.com/opentelemetry/instrument/dd_sdks/otlp_trace_export/
[dd-runtime-metrics]: https://docs.datadoghq.com/tracing/metrics/runtime_metrics/
[dd-deployment-tracking]: https://docs.datadoghq.com/tracing/services/deployment_tracking/
[otel-go]: https://opentelemetry.io/docs/languages/go/
[otel-sdk-env-vars]: https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/
[otel-go-autoexport]: https://pkg.go.dev/go.opentelemetry.io/contrib/exporters/autoexport
[otel-go-autoprop]: https://pkg.go.dev/go.opentelemetry.io/contrib/propagators/autoprop
[k8s-dependent-env-vars]: https://kubernetes.io/docs/tasks/inject-data-application/define-interdependent-environment-variables/
