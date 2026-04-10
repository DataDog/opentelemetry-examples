# OTel Demo Deployments

Helm values files for opentelemetry-demo deployments. Each file is the source of truth for a deployment — run `helm upgrade -f <values-file>` to apply.

**Chart**: `open-telemetry/opentelemetry-demo` (requires patching `_pod.tpl` to replace `tpl (toYaml $allEnvs)` with `toYaml $allEnvs` to avoid template errors with `$()` env var expansion)

## Deployments

| File | Cluster | Namespace | Env | Site | Stats Computation |
|------|---------|-----------|-----|------|-------------------|
| `values-otel-demo-staging.yaml` | opamp-demo | default | `otel-demo` | datad0g.com | spanmetrics connector → otlphttp |
| `values-otel-datadogconnector.yaml` | otel-demo | otel-datadogconnector | `otel-datadogconnector` | datad0g.com | datadog/connector → datadog/exporter |
| `values-otel-otlphttp.yaml` | otel-demo | otel-otlphttp | `otel-otlphttp` | datad0g.com | instrumentation_metrics_calc via otlphttp header |
| `values-otel-demo-us5.yaml` | otel-demo | us5-prod-test | `otel-demo-us5` | us5.datadoghq.com | spanmetrics connector → otlphttp |

## Collector Configs (live)

### `otel-demo` (staging, spanmetrics connector → otlphttp)

```yaml
connectors:
  spanmetrics:
    aggregation_temporality: AGGREGATION_TEMPORALITY_DELTA
    add_resource_attributes: true
    histogram:
      exponential: {}
      unit: s
    dimensions: [...] # env, version, status code, host, peer, container, operation, resource tags

exporters:
  otlphttp:
    compression: zstd
    endpoint: http://otlp.${env:DD_SITE}
    headers:
      dd-api-key: ${env:DD_API_KEY}
      dd-otel-metric-config: '{"trace_metrics": {"instrumentation_metrics_calc": false, "namespace": "traces.span.metrics."}, "raw_instrumentation_metrics_drop": false, "resource_attributes_as_tags": true, "instrumentation_scope_metadata_as_tags": true}'
    metrics_endpoint: http://otlp.${env:DD_SITE}/api/v2/otlpmetrics

processors:
  transform/set_env:
    trace_statements:
    - set(resource.attributes["deployment.environment.name"], "otel-demo")
    # same for metric_statements and log_statements

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection]
      exporters: [otlphttp, spanmetrics]
    metrics:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection, cumulativetodelta]
      exporters: [otlphttp]
    metrics/spanmetrics:
      receivers: [spanmetrics]
      processors: [transform/set_env]
      exporters: [otlphttp]
    logs:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection]
      exporters: [otlphttp]
```

### `otel-datadogconnector` (datadog/connector → datadog/exporter)

```yaml
connectors:
  datadog/connector:
    traces:
      compute_stats_by_span_kind: true
      peer_tags_aggregation: true

exporters:
  datadog/exporter:
    api:
      site: ${env:DD_SITE}
      key: ${env:DD_API_KEY}
  debug: {}

processors:
  transform/set_env:
    trace_statements:
    - set(resource.attributes["deployment.environment.name"], "otel-datadogconnector")
    # same for metric_statements and log_statements

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection]
      exporters: [datadog/exporter, datadog/connector]
    metrics:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection, cumulativetodelta]
      exporters: [datadog/exporter]
    metrics/datadogconnector:
      receivers: [datadog/connector]
      processors: [transform/set_env]
      exporters: [datadog/exporter, debug]
    logs:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection]
      exporters: [datadog/exporter]
```

### `otel-otlphttp` (no connector, instrumentation_metrics_calc via header)

```yaml
exporters:
  datadog/exporter:
    api:
      site: ${env:DD_SITE}
      key: ${env:DD_API_KEY}
  otlphttp:
    compression: zstd
    endpoint: http://otlp.${env:DD_SITE}
    headers:
      dd-api-key: ${env:DD_API_KEY}
      dd-otel-metric-config: '{"trace_metrics": {"instrumentation_metrics_calc": true, "namespace": "traces.span.metrics."}, "raw_instrumentation_metrics_drop": false, "resource_attributes_as_tags": true, "instrumentation_scope_metadata_as_tags": true}'
    metrics_endpoint: http://otlp.${env:DD_SITE}/api/v2/otlpmetrics
  debug: {}

processors:
  transform/set_env:
    trace_statements:
    - set(resource.attributes["deployment.environment.name"], "otel-otlphttp")
    # same for metric_statements and log_statements

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection]
      exporters: [datadog/exporter]
    metrics:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection, cumulativetodelta]
      exporters: [otlphttp]
    logs:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection]
      exporters: [otlphttp]
```

### `otel-demo-us5` (spanmetrics connector → otlphttp, us5 prod)

```yaml
connectors:
  spanmetrics:
    aggregation_temporality: AGGREGATION_TEMPORALITY_DELTA
    add_resource_attributes: true
    histogram:
      exponential: {}
      unit: s
    dimensions: [...] # same as otel-demo staging

exporters:
  otlphttp:
    endpoint: https://otlp.us5.datadoghq.com
    headers:
      dd-api-key: ${env:DD_API_KEY}
      dd-otel-metric-config: '{"trace_metrics": {"instrumentation_metrics_calc": true, "namespace": "traces.span.metrics."}, "raw_instrumentation_metrics_drop": false, "resource_attributes_as_tags": true, "instrumentation_scope_metadata_as_tags": true}'
    metrics_endpoint: https://otlp.us5.datadoghq.com/api/v2/otlpmetrics

processors:
  transform/set_env:
    trace_statements:
    - set(resource.attributes["deployment.environment.name"], "otel-demo-us5")
    # same for metric_statements and log_statements

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection]
      exporters: [otlphttp, spanmetrics]
    metrics:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection, cumulativetodelta]
      exporters: [otlphttp]
    metrics/spanmetrics:
      receivers: [spanmetrics]
      processors: [transform/set_env]
      exporters: [otlphttp]
    logs:
      receivers: [otlp]
      processors: [k8sattributes, transform/set_env, resourcedetection]
      exporters: [otlphttp]
```

## Deploy commands

```bash
# Patch the chart first (one-time)
helm pull open-telemetry/opentelemetry-demo --version 0.40.2 --untar --untardir /tmp/otel-demo-chart
sed -i '' 's/{{- tpl (toYaml \$allEnvs) \. }}/{{- toYaml $allEnvs }}/' /tmp/otel-demo-chart/opentelemetry-demo/templates/_pod.tpl

# otel-demo (opamp-demo cluster, default namespace)
helm upgrade otel-demo-recommended open-telemetry/opentelemetry-demo \
  --kube-context arn:aws:eks:us-east-1:217139788599:cluster/opamp-demo \
  --namespace default -f values-otel-demo-staging.yaml

# datadogconnector (otel-demo cluster)
helm upgrade otel-demo-ddc /tmp/otel-demo-chart/opentelemetry-demo \
  --kube-context arn:aws:eks:us-east-1:217139788599:cluster/otel-demo \
  --namespace otel-datadogconnector -f values-otel-datadogconnector.yaml

# otlphttp (otel-demo cluster)
helm upgrade otel-demo-otlphttp /tmp/otel-demo-chart/opentelemetry-demo \
  --kube-context arn:aws:eks:us-east-1:217139788599:cluster/otel-demo \
  --namespace otel-otlphttp -f values-otel-otlphttp.yaml

# us5 prod (otel-demo cluster)
helm upgrade otel-demo-us5 /tmp/otel-demo-chart/opentelemetry-demo \
  --kube-context arn:aws:eks:us-east-1:217139788599:cluster/otel-demo \
  --namespace us5-prod-test -f values-otel-demo-us5.yaml
```

## Secrets

Each namespace needs a `demo-secrets` secret with key `dd-api-key`:
```bash
kubectl create secret generic demo-secrets --from-literal=dd-api-key=<API_KEY> -n <namespace>
```

## RBAC

The first deployment creates the `otel-collector` ClusterRole. Additional deployments in other namespaces need a ClusterRoleBinding:
```bash
kubectl create clusterrolebinding otel-collector-<ns> --clusterrole=otel-collector --serviceaccount=<ns>:otel-collector
```
