# Game of Life

We have implemented Conway’s Game of Life in order to create a live demo application that we can instrument and analyze with OpenTelemetry. New members of the OpenTelemetry team will be able to experiment with this project in order to understand the flow of data from a host to the Datadog backend, as well as test various OpenTelemetry configurations.

This project is written in Go and consists of a webapp and a gRPC server. The webapp client sends a POST request with user parameters to the webapp server. The webapp server then sends the user parameters to the gRPC server in a RPC request, which processes the request by simulating the Game of Life and returns the result board. The webapp server then processes the result board and translates it to ASCII to be returned and displayed by the client.

Both services are currently instrumented with OTel Traces and Logs, and the OTel Metrics instrumentation is currently in progress.

## Running locally

To start the webapp and HTTP server, run the command:
```
go run webapp/webapp.go
```

To start the gRPC server, run the command:
```
go run server/server.go
```

To view the webapp client, navigate to http://localhost:8080/.

Input boards need to be in 2D array format, such that each array element represents a new row in the board.
For example, `[[1,1],[1,0],[0,1]]` represents the board:
```
1 1
1 0
0 1
```

## Sending telemetry data to local collector

To test this project with a local OTel Collector and Datadog Exporter setup, follow these steps:

1. Modify [logging.go](logging/logging.go) with your desired output path, and update the filelog receiver in [collector_config.yml](example/collector_config.yml) to match the updated path.
2. Run [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib) with your `DD_API_KEY` and [collector_config.yml](example/collector_config.yml).
3. Run the webapp and gRPC server with the `OTEL_SERVICE_NAME` environment variable set. Examples: `game-of-life-webapp` and `game-of-life-server`.
4. Send a request from the webapp client. The collector will output logs for the request traces and logs, and telemetry data will appear in your Datadog org.
