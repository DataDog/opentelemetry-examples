# Game Controller
This project is a simple Express.js application that uses the OpenTelemetry API for tracing and axios for HTTP requests.


## Installation
* Node.js (v12 or later)
* npm (v6 or later)


```bash
npm install
```

## Run the server

```bash
opentelemetry-instrument --service_name controller --logs_exporter otlp node game_controller.js
```

