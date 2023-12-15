## Calendar Application
```
GET /calendar
Returns a random date in 2022.
```

Request
curl 'http://localhost:8080/calendar'

Response
{"date":"3/22/2022"}


## Docker Compose

Retrieve API_KEY from datadoghq, and expose same on Shell

```
export DD_API_KEY=xx
```

Bring up java & py services with DD agent. Java service is instrumented with otel, py service is instrumented with datadog

```
docker compose -f docker-compose-otel-java-dd-py.yaml up
```

Bring up java & py services with DD agent. Java service is instrumented with dd-trace, py service is instrumented with otel

```
docker compose -f docker-compose-dd-java-otel-py.yaml up
```

Bring up two sets of java and py services with DD agent. One set is instrumented with dd-trace, the other set is instrumented with otel

```
docker compose -f docker-compose-dd-otel-both.yaml up
```