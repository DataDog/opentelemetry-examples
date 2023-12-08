# Random Date API
This directory consists of two examples of .NET applications, one instrumented with [dd-trace-dotnet](https://github.com/DataDog/dd-trace-dotnet) and the other with [OTel .NET](https://opentelemetry.io/docs/instrumentation/net/). Both applications expose the following API:

```
GET /randomdate
Returns a random date.
```

```
Request:
curl 'http://localhost:5077/randomdate'

Response:
"2020-09-09T00:00:00"
```


## Docker Compose

Retrieve your API_KEY from datadoghq, and expose your key through a shell env varabile:

```
export DD_API_KEY=xx
```

The following command will bring up the .NET service alongside the Datadog Agent. The .NET service is instrumented with dd-trace.
```
docker compose -f docker-compose-dd.yaml up
```

The following command will bring up the .NET service alongside the Datadog Agent. The .NET service is instrumented with OTel.
```
docker compose -f docker-compose-otel.yaml up
```

The following command will bring up two .NET services alongside the Datadog Agent. One .NET service is instrumented with OTel and one is instrumented with dd-trace
```
docker compose -f docker-compose-dd-otel.yaml up
```



