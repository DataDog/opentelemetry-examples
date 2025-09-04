# Game Controller
This project is a Spring Boot application that uses the OpenTelemetry API for tracing and RestTemplate for HTTP requests.


## Installation
* Java 17 or later
* Gradle (wrapper included)


```bash
./gradlew build
```

## Run the server

### With OpenTelemetry
```bash
./run-otel-local.sh
```

### With Datadog
```bash
./run-dd-local.sh
```

### Docker
```bash
docker build -t game-controller .
docker run -p 5002:5002 game-controller
```

