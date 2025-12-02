# Calendar Application

```
GET /calendar
Returns a random date within the current year.
```

```
Request:
curl '<http://localhost:9090/calendar>'

Response:
{"date":"3/22/2022"}
```

## Running locally

To build the application locally, run:
```
./gradlew build
```

You can use the script `run-otel-local.sh` to test the application locally with OTel SDK.
You can use the script `run-dd-local.sh` to test the application locally with DD SDK.

## Docker Compose

In all 3 setups, retrieve API_KEY and SITE from the Datadog app, and expose them on the shell:

```
export DD_API_KEY=xx
export DD_SITE=xx
```

**Setup 1: OTel SDK + OTel Collector:**

Bring up otel-collector & Java calendar service:

```
docker compose -f deploys/docker/docker-compose-otelcol.yml up
```

In this setup, the Collector has been configured to use the probabilistic sampler processor. This
means that only a percentage of all traces will pass through the Collector and be exported to
Datadog. The current setting for the sampler is 30% and can be adjusted in
`src/main/resources/otelcol-config.yaml`.

You can use the following bash command in a terminal to send 10 requests to the application:

    for n in {1..   10};
    do
        curl 'http://localhost:9090/calendar'
    done

After this command, we should expect to see 3 requests on average exported to Datadog.

**Setup 2: OTel SDK + Datadog Agent (OTLP ingest):**

Bring up agent & Java calendar service:

```
docker compose -f deploys/docker/docker-compose-otlp-ingest.yml up
```

**Setup 3: Datadog SDK (OTel API) + Datadog Agent:**

Bring up agent & Java calendar service:

```
docker compose -f deploys/docker/docker-compose-dd-sdk.yml up
```

The Datadog Java SDK can be used via the OTel API. However, currently, only traces can be directly
exported using this method. This example collects logs indirectly using the Datadog Agent's [Docker Log collection](https://docs.datadoghq.com/containers/docker/log/?tab=containerinstallation) feature. Metrics are simply dropped.

**Note:** If you choose to use the OTel SDK with the Agent's log collection (a combination not demonstrated here), log/trace correlation may not work OOTB. This is due to the fact that the OTel Java tracer automatically injects the keys `trace_id` and `span_id` (which conforms with [OTel conventions](https://opentelemetry.io/docs/specs/otel/compatibility/logging_trace_context/)), but the Datadog backend expects keys that conform to DD conventions (e.g. `dd.trace_id`, `dd.span_id`). In that situation, in order for trace/log correlation to work, the `trace_id` key needs to be manually added to "Preprocessing for JSON logs" at <https://app.datadoghq.com/logs/pipelines>.

## Build multi-platform images

In order to build multi-platform container images, we will use the `docker buildx` command.

Calendar app with OTel SDK:

```shell
docker buildx build \
--platform linux/amd64,linux/arm64 \
--tag datadog/opentelemetry-examples:calendar-java-20251202 \
--file ./deploys/Dockerfile.otel \
--push .
```

Calendar app with DataDog SDK:

```shell
docker buildx build \
--platform linux/amd64,linux/arm64 \
--tag datadog/opentelemetry-examples:calendar-java-dd-20251202 \
--file ./deploys/Dockerfile.dd \
--push .
```

## Kubernetes

Install calendar in K8s with OTel SDK

```
helm install -n otel-ingest calendar-otel-java ./deploys/calendar/ --set image.repository=datadog/opentelemetry-examples --set image.tag=calendar-java-20251202,node_group=ng-1
```

Install calendar in K8s with DD SDK

```
helm install -n otel-ingest calendar-dd-java ./deploys/calendar-dd/ --set image.repository=datadog/opentelemetry-examples --set image.tag=calendar-java-dd-20251202,node_group=ng-1
```
