# Trace/Log Correlation Application
This project consists of a Go client and server instrumented with Opentelemetry with the datadog-agent.

No actions are required other than spinning up the containers (see Docker Compose Section), as the client will automatically make `GET /inject` requests every 30 seconds.

The client and server spans will be part of a distributed trace, and both client and server produce a json log which is injected with the 128 bit trace_id.

**_Note:_** the trace_id and span_id are injected in the `dd.trace_id` and `dd.span_id` keys, as these are the keys in which the backend currently expects to see the trace_id/span_id [documentation](https://docs.datadoghq.com/tracing/troubleshooting/correlated-logs-not-showing-up-in-the-trace-id-panel/?tab=jsonlogs#:~:text=trace%20ID%20is-,dd.trace_id,-and%20verify%20that).

## Docker Compose
Retrieve your API_KEY from datadoghq, and expose your key on the shell:
```
export DD_API_KEY=xx
```

Bring up the client, server & datadog-agent:
```
docker-compose build
docker compose up
```

Spin down the client, server & datadog-agent:
```
docker compose down || Ctrl+C
```
