# Adapt applications to DDOT

Use this guide to install or adapt application instrumentation so that it sends OpenTelemetry telemetry through a
DDOT Collector. The flow applies to containers on Kubernetes without OpenTelemetry Operator injection and to Docker
Compose. For Kubernetes cluster telemetry and Operator-managed injection, use the [Kubernetes DDOT guide][k8s-ddot].

This guide covers Java, Node.js, Python, .NET, and Go. Support differs by language and SDK version. Enable and claim
only the signals that you verify for the exact runtime and SDK version you deploy.

> [!TIP]
> **Using a coding agent?** Give it this prompt from the repository that owns the application and its deployment:
>
> ```text
> Adapt this application's instrumentation to DDOT by following
> https://raw.githubusercontent.com/DataDog/opentelemetry-examples/main/guides/ddot/ADAPT-APPLICATIONS.md
> Inspect the current setup first. Preserve intentional choices, report conflicts before changing them, and verify
> every enabled signal through the Collector and in Datadog.
> ```

Follow one flow for every deployment:

1. [Inspect the current setup](#1-inspect-the-current-setup).
2. [Install and activate the SDK where needed](#2-install-and-activate-the-sdk-where-needed).
3. [Configure signals, identity, and destinations](#3-configure-signals-identity-and-destinations).
4. [Verify and report](#4-verify-and-report).

## Validation target versions

The repository examples use the following pins. The [DDOT Kubernetes values][k8s-values] own the current injected SDK
image pins. The [Astronomy Store][astronomy-store] owns its runtime and build-time Go pins. Record the exact resolved
package and image digest in each validation report because preview image tags can move.

| Language | Repository runtime | Manual-install pin | Operator injection image | OTLP trace transport for this pin |
|---|---|---|---|---|
| Java | Java 17 | Java agent `1.68.0` from the injection image | `datadog/dd-otel-autoinstrumentation-java:1.68.0-preview` | OTLP/HTTP protobuf or gRPC |
| Node.js | Node.js 24 | `dd-trace@6.20.0` | `datadog/dd-otel-autoinstrumentation-js:6.20-preview` | OTLP/HTTP JSON |
| Python | Python 3.12 | `ddtrace==4.16.1` | `datadog/dd-otel-autoinstrumentation-python:4.16.1-preview` | OTLP/HTTP protobuf |
| .NET | .NET 8 | Tracer `3.56.0` from the injection image | `datadog/dd-otel-autoinstrumentation-dotnet:3.56.0-preview` | OTLP/HTTP JSON or protobuf |
| Go | Go 1.26 | Orchestrion `v1.15.0` and `dd-trace-go/v2` `v2.13.0-dev.1` | Not applicable | OTLP/HTTP protobuf |

The table describes this repository's validation targets. It does not promise the same behavior for other SDK,
runtime, operating system, or architecture combinations. Consult the [OTLP trace-export support matrix][dd-otlp-export]
before changing a pin. Preserve an older working SDK until its replacement has been tested.

## 1. Inspect the current setup

Inspect the application, image build, process command, and deployment configuration before editing them. Record:

- the language runtime, operating system, architecture, package manager, lockfile, and resolved SDK versions;
- how instrumentation starts: a runtime agent, loader, wrapper command, CLR profiler, compile-time tool, or application
  code;
- dependencies on the OpenTelemetry API, SDK, auto-instrumentation packages, exporters, and Datadog SDK;
- application-created `TracerProvider`, `MeterProvider`, or `LoggerProvider` instances and their processors, readers,
  and exporters;
- `OTEL_*` and `DD_*` settings from application configuration, container environment, Compose files, Kubernetes
  manifests, ConfigMaps, Secrets, image `ENV` instructions, and startup scripts;
- every Collector or Agent endpoint, protocol, credential source, and network path;
- file or container-log collection, direct log submission, DogStatsD metrics, and runtime-metrics collection;
- Kubernetes Operator annotations and Datadog admission labels when the same manifests can run on Kubernetes.

Classify the application after the inspection:

| Starting point | Action |
|---|---|
| No instrumentation | Install and activate one DDOT SDK. |
| Upstream OpenTelemetry | Keep API instrumentation. Identify the owner of each provider and exporter before replacing any SDK setup. |
| Datadog instrumentation | Reuse the installed Datadog SDK when its version supports the required OTLP signals. Change export and semantics deliberately. |
| Mixed instrumentation | Stop and report which components can instrument the same libraries or own the same global provider. Resolve each overlap before enabling another loader. |

Preserve explicit disables such as `OTEL_SDK_DISABLED=true`, `OTEL_*_EXPORTER=none`, and
`DD_*_OTEL_ENABLED=false`. Preserve existing exporters and application-owned providers until the owner confirms the
migration. Report conflicting shared and signal-specific endpoints because a signal-specific setting takes precedence.

Check for duplicate activation before installing anything. Examples include two Java agents, `ddtrace-run` plus
`import ddtrace.auto`, two Node.js initialization hooks, a CLR profiler plus another injection mechanism, and a Go
binary compiled by both Orchestrion and another instrumentation compiler. On Kubernetes, an application must not use
both Operator injection and Datadog admission-controller injection.

## 2. Install and activate the SDK where needed

Use the application's existing package manager and lockfile. Pin the SDK version, rebuild the image, and record the
resolved version. A package dependency alone does not activate automatic instrumentation.

### Java

Copy the pinned `dd-java-agent.jar` from the validation image into the runtime image:

```dockerfile
COPY --from=datadog/dd-otel-autoinstrumentation-java:1.68.0-preview \
  /javaagent.jar /opt/datadog/dd-java-agent.jar
```

Attach it before the JVM starts:

```text
java -javaagent:/opt/datadog/dd-java-agent.jar -jar application.jar
```

`JAVA_TOOL_OPTIONS=-javaagent:/opt/datadog/dd-java-agent.jar` is equivalent when the existing launcher cannot be
changed. Append to existing JVM options. Do not attach a second Java tracing agent. Applications that create manual
spans should depend on the OpenTelemetry API only; the Datadog Java agent supplies the provider. Follow the
[Java tracing setup][java-setup] and [OpenTelemetry API support][dd-otel-api] for supported API versions.

### Node.js

Install the pinned `dd-trace` package as a production dependency:

```text
npm install --save-exact dd-trace@6.20.0
```

Load and initialize it before instrumented modules:

```text
node --require dd-trace/init app.js
```

ES modules on Node.js 20.6 or later also need the loader hook; `--import dd-trace/initialize.mjs` combines the loader
and initialization. Preserve other `NODE_OPTIONS` entries. Frameworks with their own entry point, including Next.js,
need startup-level initialization. Use one initialization path. Applications that emit OpenTelemetry logs also need
`@opentelemetry/api-logs`; applications that use the metrics API need `@opentelemetry/api`. Pin those API packages to
versions compatible with the application and `dd-trace`, and record the resolved versions. See the
[Node.js tracing setup][node-setup] and [OpenTelemetry API support][dd-otel-api].

### Python

Install the pinned `ddtrace` package:

```text
python -m pip install ddtrace==4.16.1
```

Start the application through its wrapper:

```text
ddtrace-run python app.py
```

An early `import ddtrace.auto` is an alternative when application configuration must be loaded in code before the
tracer. Use one activation path. Keep `opentelemetry-api` for API instrumentation. The pinned Python SDK requires
`opentelemetry-sdk` and the signal's `opentelemetry-exporter-otlp-proto-http` package for its OpenTelemetry metrics and
logs pipelines. Pin the OpenTelemetry API, SDK, and exporter to the same compatible release in the application's
lockfile when those signals are enabled. See the [Python tracing setup][python-setup] and
[OpenTelemetry API support][dd-otel-api].

### .NET

Copy the pinned Datadog .NET tracer files from the validation image into the runtime image:

```dockerfile
COPY --from=datadog/dd-otel-autoinstrumentation-dotnet:3.56.0-preview \
  /autoinstrumentation /opt/datadog
```

Activate the CLR profiler before `dotnet` starts:

```text
CORECLR_ENABLE_PROFILING=1
CORECLR_PROFILER={846F5F1C-F9AE-4B07-969E-05C26BC060D8}
CORECLR_PROFILER_PATH=/opt/datadog/linux/Datadog.Trace.ClrProfiler.Native.so
DD_DOTNET_TRACER_HOME=/opt/datadog
```

Use the native library path that matches the image architecture and libc. Alpine images use the musl build. Limit
these variables to the instrumented process or container. The profiler supplies automatic instrumentation; existing
code can continue to use `System.Diagnostics.Activity`, `System.Diagnostics.Metrics`, and supported logging APIs.
Follow the [.NET tracing setup][dotnet-setup] and [OpenTelemetry API support][dd-otel-api].

### Go

Go automatic instrumentation is a build-time change. Install the pinned Orchestrion release and Datadog integration in
the build stage:

```text
go install github.com/DataDog/orchestrion@v1.15.0
go get github.com/DataDog/dd-trace-go/v2@v2.13.0-dev.1 \
  github.com/DataDog/dd-trace-go/orchestrion/all/v2@v2.13.0-dev.1
```

Run `orchestrion pin` in the Go module, commit the resulting `go.mod`, `go.sum`, and `orchestrion.tool.go` changes, then
compile and test through Orchestrion:

```text
orchestrion go test ./...
orchestrion go build -o application .
```

`orchestrion pin` adds the Datadog integration imports. Review `orchestrion.tool.go` so it does not also activate an
upstream OpenTelemetry compile-time integration. Use the pinned Go version required by the resolved `dd-trace-go/v2`
module. When the application emits OpenTelemetry metrics or logs, follow [OpenTelemetry API support][dd-otel-api] for
the required `ddtrace/opentelemetry/metric` or `ddtrace/opentelemetry/log` import and provider initialization. Runtime
environment changes cannot instrument a Go binary that was built without the required integration.

## 3. Configure signals, identity, and destinations

### Start from the signals the application owns

Create a signal inventory before setting enablement variables:

| Signal | Evidence required before enablement |
|---|---|
| Traces | The SDK is active, the pinned version supports OTLP trace export, and no second tracer instruments the same libraries. |
| Application metrics | The application records OTel metrics or the SDK provides an intended metric source; required API, SDK, exporter, and provider setup is present. |
| Application logs | The application emits logs through a supported API or bridge; file tailing or direct submission will not send the same record again. |
| Runtime metrics | The pinned SDK supports the runtime and its actual transport is reachable. Treat DogStatsD and OTLP as separate paths. |

Do not infer metrics or logs support from successful trace export. Do not enable an unused signal to make the
configuration look uniform across languages.

For supported signals, set the matching DDOT switches before process startup. For the repository's reference
configuration, see `instrumentation.env` in [`values.yaml`][k8s-values].

Keep `DD_METRICS_OTEL_ENABLED` or `DD_LOGS_OTEL_ENABLED` unset when that signal is outside the application contract.
Keep an explicit `false` or `none` unless the application owner approves changing it. The switches do not install
missing packages, initialize application-owned providers, or prove that data was exported.

The Kubernetes reference values enable both switches globally because the Astronomy Store showcase exercises those
signals. This demo default applies only to workloads whose SDK supports both signals. Override an unowned or unsupported
signal in the workload's container environment, or use a separate `Instrumentation` resource whose `spec.env` omits the
switch.

### Set service identity

Set stable identity once and use the same values for every enabled signal:

```text
OTEL_SERVICE_NAME=<SERVICE_NAME>
OTEL_RESOURCE_ATTRIBUTES=service.namespace=<NAMESPACE>,service.version=<VERSION>,deployment.environment.name=<ENVIRONMENT>
```

For an existing Datadog SDK setup, inspect `DD_SERVICE`, `DD_ENV`, `DD_VERSION`, and `DD_TAGS` before adding the
OpenTelemetry identity. Do not leave conflicting values in the same process. Keep both naming schemes aligned when
native Datadog, DogStatsD, or runtime-metrics paths still consume the `DD_*` settings:

```text
DD_SERVICE=<SERVICE_NAME>
DD_ENV=<ENVIRONMENT>
DD_VERSION=<VERSION>
```

Merge unrelated `DD_TAGS` with the intended resource attributes. Remove an existing `DD_*` identity setting only after
verifying that no remaining signal or Datadog feature consumes it.

Merge with existing resource attributes. Do not erase attributes supplied by the runtime or deployment platform.
On Kubernetes, the Collector adds Kubernetes resource attributes; follow the [Kubernetes DDOT guide][k8s-ddot]
instead of copying pod metadata into application configuration. See [unified service tagging][unified-tagging] for how
OpenTelemetry resource attributes map to Datadog service, environment, and version tags.

### Point each signal at the Collector

Prefer an OTLP/HTTP receiver on port `4318`. The shared endpoint is a base URL. OTLP exporters append `/v1/traces`,
`/v1/metrics`, or `/v1/logs`:

```text
OTEL_EXPORTER_OTLP_ENDPOINT=http://<COLLECTOR_HOST>:4318
```

Use a signal-specific endpoint when a pinned SDK requires it or when signals use different destinations:

```text
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://<COLLECTOR_HOST>:4318/v1/traces
OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://<COLLECTOR_HOST>:4318/v1/metrics
OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://<COLLECTOR_HOST>:4318/v1/logs
```

Signal-specific endpoints must include the signal path. A signal-specific value overrides the shared endpoint. The
pinned Go setup needs `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` because its trace exporter does not honor the shared
endpoint. Keep credentials in Secrets and preserve existing `OTEL_EXPORTER_OTLP_*_HEADERS` values.

Configure a protocol supported by the pinned SDK and Collector receiver. For the repository pins:

```text
# Java, Python, Go, and .NET 3.45 or later
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf

# Node.js traces
OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http/json
```

Node.js metrics and logs support OTLP/HTTP protobuf at the pinned version, so use signal-specific protocol settings
when traces, metrics, and logs are enabled together. Do not change an endpoint from port `4317` to `4318` without also
changing `grpc` to an HTTP protocol. See the [OTLP exporter specification][otel-otlp-config] and the
[Datadog SDK OTLP matrix][dd-otlp-export].

Datadog requires delta metrics. Set the preference when the SDK supports it:

```text
OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE=delta
```

If another destination requires cumulative metrics, preserve cumulative export and convert only the Datadog branch
with the Collector's `cumulative_to_delta` processor. Account for the processor's state and ensure all points in a
time series reach the same Collector instance. The [Kubernetes DDOT guide][k8s-ddot] owns the reference Collector
pipeline.

### Check Collector and Agent requirements

The Collector must expose the configured OTLP receiver and route each enabled signal through a pipeline with a Datadog
exporter. Put the Datadog connector before trace sampling when the deployment uses it to compute APM trace metrics.
For Kubernetes, follow the [Kubernetes DDOT guide][k8s-ddot]. For other environments, follow the
[Datadog Collector setup][dd-collector-setup].

A Datadog Agent is optional for the basic OTLP path from the SDK to a Collector. Some Datadog features use an Agent for
Remote Configuration, Live Debugger, Dynamic Instrumentation, or DogStatsD. Set `DD_AGENT_HOST` only when a reachable
Agent provides a feature the application uses. If the SDK's runtime metrics use DogStatsD, expose UDP `8125` and keep
that Agent path. The Datadog SDK [runtime-metrics guide][dd-runtime-metrics] owns the language-specific defaults and
metric names. If the application sends OTLP directly to an Agent, enable the Agent's OTLP receiver and use Agent 7.48.0
or later for logs. Avoid sending the same signal through both Collector and Agent paths.

### Adapt existing providers and exporters deliberately

When upstream OpenTelemetry code creates a provider or exporter, map ownership signal by signal:

- Keep API calls and instrumentation libraries that the DDOT SDK supports.
- Keep an application-owned provider and exporter until the owner approves moving that signal to the DDOT-managed
  provider. Do not register two global providers.
- When migrating a provider, remove its SDK initialization and exporter only after the DDOT SDK packages, imports,
  activation, and signal support are in place. Follow [OpenTelemetry API support][dd-otel-api] for language-specific
  migration details.
- Preserve secondary exporters when dual export is intentional. Verify that they do not send a second copy to
  Datadog.

When adapting existing Datadog instrumentation, reuse its loader or profiler. Configure OTLP export and verify the
new semantics. Disable native trace export to the Agent only after OTLP trace delivery succeeds. Review direct log
submission, file tailing, DogStatsD, and runtime metrics independently because they do not all follow the trace
exporter.

## 4. Verify and report

Build and run the resulting container with the same command and runtime that production uses. Generate identifiable
traffic and test telemetry. Use a unique validation value, such as a test-run resource attribute, and emit a known
span, metric point, and log record for each enabled signal.

Verify all of the following:

1. **Activation:** process arguments, startup logs, or SDK diagnostics prove that one intended agent, loader, profiler,
   or compiled integration is active.
2. **Effective configuration:** inspect the running container's environment and SDK diagnostics. Resolve shared and
   signal-specific endpoint precedence. Redact headers and credentials from retained evidence.
3. **Collector receipt:** Collector telemetry and temporary debug output show the unique test data for every enabled
   signal. A healthy receiver without the test record is insufficient.
4. **Datadog receipt:** find the same test data in Trace Explorer, Metrics Explorer, and Log Explorer as applicable.
   Check `service`, `env`, `version`, and expected semantic-convention attributes.
5. **Runtime metrics:** when enabled, find runtime-specific metric names in Datadog and record whether they traveled by
   OTLP or DogStatsD. Successful application metrics do not prove runtime metrics.
6. **No duplicates:** confirm one expected trace tree per request, one copy of the marked log record, and the expected
   metric series and points. Check both Collector and Agent paths.
7. **Idempotence:** run the coding agent against the adapted repository again. It should report the setup as satisfied
   without changing dependencies, loaders, providers, exporters, or deployment configuration.

Record a result for every attempted application:

| Field | Required evidence |
|---|---|
| Source | Application revision, guide revision, fixture revision, and unmodified or adapted starting state |
| Versions | Runtime, operating system, architecture, SDK package and image digest, Collector, and Agent |
| Activation | Exact build and startup mechanism and evidence that it loaded |
| Configuration | Effective non-secret endpoint, protocol, enabled signals, resource identity, and provider ownership |
| Traces | Collector evidence, Datadog evidence, semantic attributes, and duplicate check |
| Application metrics | Collector evidence, Datadog evidence, temporality, and duplicate check |
| Application logs | Collector evidence, Datadog evidence, correlation, and duplicate check |
| Runtime metrics | Enabled state, transport, metric names, Datadog evidence, or explicit reason it was not enabled |
| Conflicts | Deliberate disables, alternative exporters, existing providers, loaders, and decisions preserved |
| Outcome | `passed`, `failed`, `blocked`, or `untested`, with the reason |

A blocked or untested signal is not a pass. Limit the final claim to the recorded language, versions, runtime,
operating system, architecture, deployment type, and signals.

[astronomy-store]: ../../apps/astronomy-store/
[dd-collector-setup]: https://docs.datadoghq.com/opentelemetry/setup/collector_exporter/
[dd-otel-api]: https://docs.datadoghq.com/opentelemetry/instrument/dd_sdks/api_support/
[dd-otlp-export]: https://docs.datadoghq.com/opentelemetry/instrument/dd_sdks/otlp_trace_export/
[dd-runtime-metrics]: https://docs.datadoghq.com/tracing/metrics/runtime_metrics/
[dotnet-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/dotnet-core/
[java-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/java/
[k8s-ddot]: ../kubernetes/configuration/opentelemetry-kube-stack/ddot/ADAPT-EXISTING-CONFIG.md
[k8s-values]: ../kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml
[node-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/nodejs/
[otel-otlp-config]: https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/
[python-setup]: https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/python/
[unified-tagging]: https://docs.datadoghq.com/getting_started/tagging/unified_service_tagging/
