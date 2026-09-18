# Go product validation over OTLP

This service validates `dd-trace-go/v2` 2.10.1 with the Datadog
OpenTelemetry tracer-provider bridge. Traces use standard `OTEL_*`
configuration and OTLP HTTP/protobuf to reach standalone DDOT. The Datadog
Agent remains available for Continuous Profiling and PostgreSQL DBM.

The validation endpoints generate deterministic CPU, heap, goroutine,
database, error, and manual OpenTelemetry-span activity. Deploy with
`k8s/go-operator-validation.yaml` and `k8s/go-ddot-collector.yaml`, then run
`scripts/run-go-product-scenario.sh` through a port-forward on local port
15003. Run one named scenario at a time so every matrix row gets a clean,
fixed evidence window.

Build requires Go 1.25 or later.
