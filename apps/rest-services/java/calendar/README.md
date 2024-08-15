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

The application has been configured to use the probabilistic sampler processor. This means that only a percentage of all traces sent through the application after passing through the sampler in the pipeline will be available following the processor in the pipeline. The current setting for the sample size is 30%, therefore roughly 30% of requests will be sent out from the processor after ingesting 100% of requests.

### Usage

First run the example using either of the Docker compose commands listed below. We expect 30% of the requests sent through the pipeline to reach the end. Use the following bash command in a terminal to send 10 requests to the application:

    for n in {1..   10};
    do
        curl 'http://localhost:9090/calendar'
    done

We should expect to see roughly 30% of the 10 requests exported to our backend of choice.

## Script

We can use scripts `run-otel-local.sh` to test application locally with OTel SDK.
We can use scripts `run-dd-local.sh` to test application locally with DD SDK.

## Docker Compose

**Collector:**

Retrieve API_KEY from datadoghq, and expose same on Shell

```
export DD_API_KEY=xx
```

Bring up otel-collector & Java calendar service

```
docker compose -f deploys/docker/docker-compose-otel.yml  up
```

**OTLP ingest in the agent:**

Retrieve API_KEY from datadoghq, and expose same on Shell

```
export DD_API_KEY=xx
```

Bring up agent & Java calendar service

```
docker compose -f deploys/docker/docker-compose-dd.yml  up
```

**Note:** This app demonstrates that log/trace correlation does not work OOTB for otel users that collect logs through the agents native functionality. This is due to the fact that the OTel java tracer automatically injects the keys `trace_id` and `span_id` (which conforms with [OTel conventions](https://github.com/open-telemetry/opentelemetry-collector/blob/7b6937aacd0232c7f07f503b20ae7a8a70336914/pdata/plog/json.go#L118-L125)), but the backend expects the trace ID to be in keys that conform to DD conventions (e.g. `dd.trace_id`, `dd.span_id`).

As of now, in order for trace/log correlation to work the `trace_id` key needs to be manually added to the "Preprocessing for JSON logs" at <https://app.datadoghq.com/logs/pipelines>.

## Kubernetes

Install calendar in K8s with OTel SDK

```
helm install -n otel-ingest calendar-otel-java ./deploys/calendar/ --set image.repository=datadog/opentelemetry-examples --set image.tag=calendar-java-otel-0.1,node_group=ng-1
```

Install calendar in K8s with DD SDK

```
helm install -n otel-ingest calendar-dd-java ./deploys/calendar-dd/ --set image.repository=datadog/opentelemetry-examples --set image.tag=calendar-java-otel-0.1,node_group=ng-1
```
