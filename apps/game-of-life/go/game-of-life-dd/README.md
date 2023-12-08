# Game of Life

We have implemented Conway’s Game of Life in order to create a live demo application that we can instrument and analyze with OpenTelemetry. New members of the OpenTelemetry team will be able to experiment with this project in order to understand the flow of data from a host to the Datadog backend, as well as test various OpenTelemetry configurations.

This project is written in Go and consists of a webapp and a gRPC server. The webapp client sends a POST request with user parameters to the webapp server. The webapp server then sends the user parameters to the gRPC server in a RPC request, which processes the request by simulating the Game of Life and returns the result board. The webapp server then processes the result board and translates it to ASCII to be returned and displayed by the client. 

Both services are currently instrumented with Datadog runtime metrics.

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
