Inspired by the OpenTelemetry Demo. Demonstrates OpenTelemetry automatic instrumentation.

Intended to be used with the `guides/kubernetes/configuration/opentelemetry-kube-stack/install`

This demo app is called `astronomy-store` instead of `astronomy-shop` to avoid confusion with the `astronomy-shop` demo
app that is part of the OpenTelemetry Demo while being similar.

## Git/VCS metadata

Each service's `Dockerfile` accepts `VCS_REPOSITORY_URL_FULL`, `VCS_REF_HEAD_NAME`, `VCS_REF_TYPE`, and
`VCS_REF_HEAD_REVISION` build args (populated by `build-images` from the local git checkout) and bakes them into
the image as `DD_GIT_REPOSITORY_URL` and `DD_GIT_COMMIT_SHA` env vars, which Datadog tracers/agents use for
Source Code Integration (linking traces/errors back to the exact commit).

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