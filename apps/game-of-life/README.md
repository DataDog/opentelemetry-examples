# Game of Life

The goal of the project is to implement Conway’s Game of Life in order to create a live demo application that we can analyze with Datadog. The Game of Life was chosen because it can be configured to generate varying amounts of CPU usage and RPC data sizes for us to view and analyze performance through Datadog.

## go

Game of Life is implemented in Go with two different versions. `game-of-life-otel` is the application instrumented with [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/), and `game-of-life-dd` is the application instrumented with [dd-trace-go](https://github.com/DataDog/dd-trace-go).

## proto

This folder contains `gameoflife.proto`, a language agnostic proto file that defines the structures for data we want to serialize when sending over gRPC.
