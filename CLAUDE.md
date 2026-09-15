# CLAUDE.md

Guidance for Claude Code and other AI assistants working in this repository.

## What this repository is

Reference examples showing how Datadog supports [OpenTelemetry][otel]: instrumented sample
applications, and Collector/Agent configurations. It is a **teaching repository**. Readers copy
these files into their own clusters and compose files, so clarity and correctness of the examples
matter more than brevity, and every configuration change is a change to documentation.

There is no repository-wide build, test suite, or published artifact — each example builds and
runs on its own, usually through Docker Compose. The only CI is CodeQL
(`.github/workflows/codeql.yml`), which scans C#, Go, Java/Kotlin, JavaScript/TypeScript and
Python on pushes and pull requests against `main`. Nothing lints or renders the YAML, so
configuration changes have to be verified locally.

## Layout

| Path | Contents |
| :--- | :--- |
| `apps/` | Instrumented sample applications, one directory per example, mostly run with Docker Compose. Several ship parallel `docker-compose-dd.yaml` / `-otel.yaml` / `-dd-otel.yaml` files to contrast Datadog and OpenTelemetry instrumentation of the same app. |
| `configurations/` | Standalone Collector and Datadog Agent configuration snippets. |
| `guides/` | Prose guides. `common-mistakes.md`, and `kubernetes/` for the Kubernetes integration. |
| `.github/` | CodeQL workflow, issue and pull request templates, `CODEOWNERS` (`@DataDog/opentelemetry` owns everything). |

## Before you change anything

Read [`CONTRIBUTING.md`](CONTRIBUTING.md). The rule that matters most: **open an issue and reach a
resolution before adding a new feature or example.** Bug fixes can go straight to a pull request.
Target `main`. New examples must include documentation for running, building and deploying them.

## Working on `guides/kubernetes`

This is the most intricate part of the repository and the easiest to break. Two deployment paths
are maintained in parallel, and **a change to one usually belongs in the other**:

- `configuration/cluster-collector.yaml` and `configuration/daemonset-collector.yaml` — Helm
  **values** files for the `opentelemetry-collector` chart, not standalone collector configs. The
  collector config lives under the top-level `config:` key.
- `configuration/opentelemetry-kube-stack/values.yaml` — values for the `opentelemetry-kube-stack`
  chart, which renders one `OpenTelemetryCollector` CR per entry under `collectors:`.

### The kube-stack examples are generated

Everything under `configuration/opentelemetry-kube-stack/examples/*/rendered/` is generated from
`values.yaml` plus the per-example overlay. Never hand-edit it. After changing `values.yaml`:

```sh
cd guides/kubernetes/configuration/opentelemetry-kube-stack
make generate-examples
```

**This requires Helm 3.** Helm 4 renders these charts differently and produces ~100 files of
spurious diff. Install it alongside Helm 4 if needed:

```sh
brew install helm@3
export PATH="$(brew --prefix helm@3)/bin:$PATH"
```

The Makefile pins the chart version (`0.20.8`), so a clean tree must regenerate byte-for-byte —
run `make generate-examples` on an unmodified checkout first and confirm `git status` is clean
before trusting a regenerated diff.

### Validating a collector config without a cluster

Render the chart, pull the collector config out of the `relay` ConfigMap, and load it:

```sh
helm template otel-cluster-collector open-telemetry/opentelemetry-collector \
  --version 0.156.2 -f guides/kubernetes/configuration/cluster-collector.yaml \
  --set image.repository=otel/opentelemetry-collector-contrib --set image.tag=0.154.0
otelcol-contrib validate --config=file:<extracted-config>.yaml
```

`validate` still fails outside a cluster on the `k8s_api` resource detector, the hostmetrics
`root_path` (Linux only) and the ServiceAccount token path. Those are environment errors; what
you are checking for is schema errors, which appear as `cannot unmarshal` or `invalid keys`.

To exercise a pipeline's behaviour, serve a Prometheus exposition fixture over HTTP, point the
`prometheus` receiver at it, drop the cluster-dependent components (`resourcedetection`,
`k8s_attributes`) and swap the `datadog` exporter for `file`. That is enough to see exactly which
series and attributes would reach Datadog.

### Datadog attribute mapping

The Datadog exporter automatically maps OTel **resource** attributes to Datadog tags
(`k8s.cluster.name` → `kube_cluster_name`, `k8s.namespace.name` → `kube_namespace`, and so on —
see [opentelemetry-mapping-go][mapping]). It does **not** map **datapoint** attributes, which is
why the KSM pipelines carry explicit `transform/ksm_to_dd` processors that rename the Prometheus
labels by hand. Keep that distinction in mind before adding or removing a rename.

Remember that every series reaching the `datadog` exporter is billed as a custom metric. Anything
a pipeline emits incidentally — Prometheus' synthetic `scrape_*` series, a connector's built-in
default metric — should be filtered out deliberately rather than left to flow through.

## Conventions

- Keep explanatory comments in the example configs. They are read by users, not just maintainers,
  and they are the reason someone can copy a block and understand it.
- Match the surrounding file's comment density, indentation and naming. These YAML files vary in
  style between directories; follow the file you are in.
- Reference documentation with link definitions collected at the bottom of the Markdown file,
  matching the existing numbering or slug style in that file.
- Pin versions in examples (chart versions, image tags) rather than tracking `latest`.

[otel]: https://opentelemetry.io/
[mapping]: https://github.com/DataDog/opentelemetry-mapping-go/blob/main/pkg/otlp/attributes/attributes.go
