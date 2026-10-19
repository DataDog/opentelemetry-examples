Inspired by the OpenTelemetry Demo. Demonstrates OpenTelemetry automatic instrumentation.

Intended to be used with the `guides/kubernetes/configuration/opentelemetry-kube-stack/install`

This demo app is called `astronomy-store` instead of `astronomy-shop` to avoid confusion with the `astronomy-shop` demo
app that is part of the OpenTelemetry Demo while being similar.

## Git/VCS metadata

Each service's `Dockerfile` accepts `VCS_REPOSITORY_URL_FULL`, `VCS_REF_HEAD_NAME`, `VCS_REF_TYPE`, and
`VCS_REF_HEAD_REVISION` build args (populated by `build-images` from the local git checkout) and bakes them into
the image as `DD_GIT_REPOSITORY_URL` and `DD_GIT_COMMIT_SHA` env vars, which Datadog tracers/agents use for
Source Code Integration (linking traces/errors back to the exact commit).

In the `upstream` flow (upstream OpenTelemetry SDKs), the `ad`, `fraud-detection`, `frontend`, and
`product-catalog` images also report two OpenTelemetry [VCS semantic-convention resource attributes]
(https://opentelemetry.io/docs/specs/semconv/registry/entities/vcs/): `vcs.repository.url.full` and
`vcs.ref.head.revision`.

These can't be set with an image-level `OTEL_RESOURCE_ATTRIBUTES` env var. The OpenTelemetry Operator sets
`OTEL_RESOURCE_ATTRIBUTES` in the pod spec (with `k8s.*`, `service.*`, `deployment.environment.name`, etc.), and a
pod spec env var replaces an image `ENV` of the same name. Instead, each of these images bakes
`VCS_REPOSITORY_URL_FULL` and `VCS_REF_HEAD_REVISION` as env vars and starts through `otel-env.sh`, which runs
when the container starts, after the Operator's injection, appends the two `vcs.*` keys to the injected
`OTEL_RESOURCE_ATTRIBUTES`, then runs the container command:

- Keys already present in `OTEL_RESOURCE_ATTRIBUTES` (pod spec env, `Instrumentation` CR,
  `resource.opentelemetry.io/*` pod annotations) are not overridden.
- The script does nothing in the `ddot` flow (`DD_TRACE_OTEL_ENABLED=true`), where the Datadog SDKs read
  `DD_GIT_*` instead.
- A Deployment that sets `command:` replaces the image `ENTRYPOINT`, so it must start with
  `/usr/local/bin/otel-env.sh` to keep the `vcs.*` attributes. Setting only `args:` keeps it.
- Each service directory has its own copy of `otel-env.sh` because each image is built with its service
  directory as the build context. Keep the copies identical.

`checkout` doesn't report `vcs.*`: its runtime image (`gcr.io/distroless/static-debian12`) has no shell to run
`otel-env.sh`.