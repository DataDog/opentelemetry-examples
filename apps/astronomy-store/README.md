Inspired by the OpenTelemetry Demo. Demonstrates OpenTelemetry automatic instrumentation.

Intended to be used with the `guides/kubernetes/configuration/opentelemetry-kube-stack/install`

This demo app is called `astronomy-store` instead of `astronomy-shop` to avoid confusion with the `astronomy-shop` demo
app that is part of the OpenTelemetry Demo while being similar.

The demo includes a single-node Kafka broker available to its workloads at
`kafka:9092`.

## Run the demo

### 1. Use a clean Kubernetes cluster

Use a Kubernetes cluster with no OpenTelemetry or Datadog installation, with `kubectl` and `helm` pointed at it
(`kubectl config current-context`). An existing OpenTelemetry Operator, Datadog Agent, or Datadog Operator conflicts
with the ones installed in the next step.

### 2. Install OpenTelemetry monitoring

We recommend installing Datadog's version of OpenTelemetry with the installer of the
[Datadog OTel Kube Stack](../../guides/kubernetes/configuration/opentelemetry-kube-stack) (see its
[installation guide](../../guides/kubernetes/configuration/opentelemetry-kube-stack/INSTALL.md)). Upstream
OpenTelemetry also works: the services are instrumented with vendor-neutral OpenTelemetry, so any OpenTelemetry Operator
`Instrumentation` named `opentelemetry-kube-stack` in the `opentelemetry-operator-system` namespace, and any collector
receiving OTLP, works.

We recommend exporting the traces not only to Datadog but also to another OpenTelemetry backend, such as Jaeger, to show
that the same telemetry works with any backend:

1. Install Jaeger with [`demo-multi-o11y-backends/jaeger`](demo-multi-o11y-backends/jaeger):

   ```sh
   ./demo-multi-o11y-backends/jaeger/install
   ```

2. Install the Datadog OTel Kube Stack with the
   [`export-to-datadog-and-jaeger`](../../guides/kubernetes/configuration/opentelemetry-kube-stack/examples/export-to-datadog-and-jaeger/values.yaml)
   overlay, which adds Jaeger (`jaeger.jaeger.svc.cluster.local:4317`) to the `daemon` collector's trace exporters:

   ```sh
   cd ../../guides/kubernetes/configuration/opentelemetry-kube-stack
   ./install examples/export-to-datadog-and-jaeger/values.yaml
   ```

See [`demo-multi-o11y-backends`](demo-multi-o11y-backends) for more about this setup.

### 3. Build the container images

```sh
./build-images
```

`build-images` builds the images of the `ad`, `checkout`, `fraud-detection`, `frontend`, and `product-catalog` services,
named `<IMAGE_PREFIX>/<service>:<IMAGE_TAG>`. It also builds the `checkout` service with upstream OpenTelemetry
(`checkout/Dockerfile.upstream-otel`) as `<IMAGE_PREFIX>/checkout:<IMAGE_TAG>-upstream-otel`. It takes no arguments,
only these environment variables:

| Variable | Default | Description |
|---|---|---|
| `IMAGE_PREFIX` | `astronomy-store` | Image name prefix, typically your container registry, e.g. `ghcr.io/example/astronomy-store`. A trailing `/` is removed. |
| `IMAGE_TAG` | `latest` | Image tag. |

```sh
IMAGE_PREFIX=ghcr.io/example/astronomy-store IMAGE_TAG=v1.2.3 ./build-images
```

The script also reads the git metadata of the local checkout (remote URL, branch, commit) and passes it to the image
builds, see [Git/VCS metadata](#gitvcs-metadata). `./build-images --help` prints the usage.

The images must be available to the Kubernetes cluster's nodes (the manifests use `imagePullPolicy: IfNotPresent`):
with a local cluster such as Docker Desktop, the images built locally are directly available; otherwise, push them to a
registry the cluster can pull from (`docker push`), or load them into the cluster (e.g. `kind load docker-image`).

### 4. Deploy the app

```sh
./install-app
```

`install-app` creates the `astronomy-store` namespace, deploys `astronomy-db`, `kafka`, `ad`, `product-catalog`,
`fraud-detection`, `checkout`, and `frontend`, waits for their rollouts, then deploys the `k6-load-generator` that
generates traffic. It takes the same `IMAGE_PREFIX` and `IMAGE_TAG` environment variables as `build-images`, to deploy
the images you built, plus `ROLLOUT_TIMEOUT` (default `3m`), how long to wait for each rollout:

```sh
IMAGE_PREFIX=ghcr.io/example/astronomy-store IMAGE_TAG=v1.2.3 ./install-app
```

`install-app` deploys the `checkout` image instrumented with the DDOT SDK for Go, not its `-upstream-otel` variant.

To browse the store, run `kubectl port-forward -n astronomy-store svc/frontend 8080:8080` and open
http://localhost:8080. To remove the app, run `./uninstall-app`.

## Demo goals

The `astronomy-store` is designed to showcase the value of the DDOT Suite:

* **Fully vendor-neutral OpenTelemetry instrumentation**: the services contain no Datadog-specific code.
* **Works with both upstream OpenTelemetry and Datadog's version of OpenTelemetry (DDOT)**: the same services can be
  instrumented with the upstream OTel SDKs or with the DDOT SDKs.
* **Datadog's version of OpenTelemetry adds Datadog-specific value on top of upstream OpenTelemetry**, demoed below
  (Database Monitoring, Data Streams Monitoring, Source Code Integration).

### OpenTelemetry instrumentation across programming languages

| Service | Language | Instrumentation |
|---|---|---|
| [`ad`](ad) | Java | OTel SDK injected by the OpenTelemetry Operator (`instrumentation.opentelemetry.io/inject-java`) |
| [`product-catalog`](product-catalog) | .NET | OTel SDK injected by the OpenTelemetry Operator (`instrumentation.opentelemetry.io/inject-dotnet`) |
| [`frontend`](frontend) | Node.js | OTel SDK injected by the OpenTelemetry Operator (`instrumentation.opentelemetry.io/inject-nodejs`) |
| [`fraud-detection`](fraud-detection) | Python | OTel SDK injected by the OpenTelemetry Operator (`instrumentation.opentelemetry.io/inject-python`) |
| [`checkout`](checkout) | Go | Compile-time instrumentation in the `Dockerfile`; the Operator only injects the SDK configuration (`instrumentation.opentelemetry.io/inject-sdk`) |

Go requires compile-time instrumentation, demoed at build time in the `checkout` service's Dockerfiles:

* [`checkout/Dockerfile.upstream-otel`](checkout/Dockerfile.upstream-otel) injects the upstream OpenTelemetry Go SDK
  with [otelc](https://opentelemetry.io/docs/zero-code/go/compile-time/). `build-images` publishes it with the
  `-upstream-otel` tag suffix.
* [`checkout/Dockerfile`](checkout/Dockerfile) injects the DDOT SDK for Go with
  [Orchestrion](https://github.com/datadog/orchestrion).

### DDOT value adds

* **Datadog Database Monitoring (DBM)**: the PostgreSQL database [`astronomy-db`](astronomy-db/kubernetes.yaml) is
  monitored by the Datadog Agent, configured with the `ad.datadoghq.com/astronomy-db.checks` pod annotation. The
  database is accessed by the `ad` service, whose queries are correlated with DBM through `DD_DBM_PROPAGATION_MODE`
  set in the `Instrumentation` CR.
* **Datadog Data Streams Monitoring (DSM)**: the Kafka broker [`kafka`](kafka/kubernetes.yaml) is monitored by the
  Datadog Agent, configured with the `ad.datadoghq.com/kafka.checks` pod annotation. Kafka is used by the `checkout`
  (producer) and `fraud-detection` (consumer) services, which report data streams through `DD_DATA_STREAMS_ENABLED`
  set in the `Instrumentation` CR.
* **Datadog Source Code Integration**: each `Dockerfile` sets the `DD_GIT_REPOSITORY_URL` and `DD_GIT_COMMIT_SHA`
  env vars (see [Git/VCS metadata](#gitvcs-metadata)). Source Code Integration is typically used to demo the DDOT
  Full Host Profiler, Datadog Live Debugger, and Datadog Dynamic Instrumentation.
* ...

## Git/VCS metadata

Each service's `Dockerfile` accepts `VCS_REPOSITORY_URL_FULL`, `VCS_REF_HEAD_NAME`, `VCS_REF_TYPE`, and
`VCS_REF_HEAD_REVISION` build args (populated by `build-images` from the local git checkout) and bakes them into
the image as `DD_GIT_REPOSITORY_URL` and `DD_GIT_COMMIT_SHA` env vars, which Datadog tracers/agents use for
Source Code Integration (linking traces/errors back to the exact commit). `build-images` rewrites SSH remotes
(`git@github.com:org/repo.git`) to https URLs (`https://github.com/org/repo`), as Source Code Integration expects.

Each `Dockerfile` also sets [OCI image labels](https://specs.opencontainers.org/image-spec/annotations/), so an
image's origin can be read from the registry or with `docker inspect` without running it:

| Label | Value |
|---|---|
| `org.opencontainers.image.title` | `astronomy-store-<service>` |
| `org.opencontainers.image.description` | One-line description of the service |
| `org.opencontainers.image.source` | Repository https URL (`VCS_REPOSITORY_URL_FULL`) |
| `org.opencontainers.image.revision` | Full commit SHA (`VCS_REF_HEAD_REVISION`) |
| `org.opencontainers.image.version` | Short commit SHA (`IMAGE_VERSION`) |
| `org.opencontainers.image.created` | Commit date in UTC (`IMAGE_CREATED`), not the build time, so rebuilding the same commit produces the same labels |

Setting `title`, `version`, and `created` also overrides the labels inherited from base images (`eclipse-temurin`,
used by `ad`, otherwise labels the image `ubuntu` version `26.04`). These labels describe the image, not the running
workload: neither OpenTelemetry nor Datadog Source Code Integration reads them, which is why the `DD_GIT_*` env vars
are still needed.

We deliberately do **not** also set the OpenTelemetry [VCS semantic-convention resource attributes]
(https://opentelemetry.io/docs/specs/semconv/registry/entities/vcs/) (`vcs.repository.url.full`,
`vcs.ref.head.name`, `vcs.ref.type`, `vcs.ref.head.revision`) via an image-level `OTEL_RESOURCE_ATTRIBUTES` env
var. These pods are auto-instrumented by the OpenTelemetry Operator, and the Operator injects its own
`OTEL_RESOURCE_ATTRIBUTES` pod env var (with `k8s.*`, `service.*`, `deployment.environment`, etc.) derived from
the `Instrumentation` CR. A pod spec env var completely replaces an image-baked `ENV` of the same name rather
than merging with it, so any `vcs.*` keys set in the Dockerfile would simply be dropped — silently overwritten
by the Operator's value the moment the pod starts.

For the `ad` service specifically (instrumented with Datadog's `dd-java-agent`, not the plain OpenTelemetry Java
agent), this can't be worked around either: `dd-java-agent` doesn't support OpenTelemetry SDK-level extension
points like `ResourceProvider` (only instrumentation-library extensions), and setting `DD_TAGS` as an
alternative causes `dd-java-agent` to ignore `OTEL_RESOURCE_ATTRIBUTES` entirely — trading the `vcs.*` attributes
for the loss of `deployment.environment`, `k8s.pod.name`, `k8s.namespace.name`, `service.namespace`, and
`service.version`.

If you want `vcs.*` attributes on these services' telemetry, add them to the `Instrumentation` CR's
`spec.resource.resourceAttributes` in `values.yaml` instead, so they're merged in at injection time rather than
raced against it.