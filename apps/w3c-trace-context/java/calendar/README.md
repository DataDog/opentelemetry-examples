## Calendar Java Service

Spring Boot application that acts as the frontend service in the W3C Trace
Context propagation demo. It receives a `GET /calendar` request and forwards
it to the downstream Python calendar service, returning the result.

### Endpoints

```
GET /calendar  -> calls the Python backend and returns a random date
GET /health    -> health check
```

### Build

```bash
./gradlew build
```

### Run locally

```bash
CALENDAR_SERVICE_URL=http://localhost:9090 SERVER_PORT=8080 \
  java -javaagent:opentelemetry-javaagent.jar -jar build/libs/calendar-0.0.1-SNAPSHOT.jar
```

### Docker

See the parent directory's docker-compose files for containerised deployment.
