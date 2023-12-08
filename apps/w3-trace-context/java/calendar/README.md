## Calendar Application
```
GET /calendar
Returns a random date in 2022.
```

Request
curl 'http://localhost:9090/calendar'

Response
{"date":"3/22/2022"}


## Docker Compose

Retrieve API_KEY from datadoghq, and expose same on Shell

```
export DD_API_KEY=xx

```
Bring up otel-collector & Java calendar service

```
docker compose -f deploys/docker/docker-compose-otel.yml  up
```

