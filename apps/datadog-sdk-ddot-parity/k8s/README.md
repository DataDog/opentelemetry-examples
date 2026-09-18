# Kubernetes validation lanes

Each SDK has an isolated namespace containing the application, PostgreSQL, and
a Datadog Agent. The DDOT setup also runs standalone DDOT through the upstream
OpenTelemetry Operator. The Agent remains running for product channels that do
not use the OTLP trace path.

| SDK | Namespace | Application manifest | DDOT manifest | Port-forward | Workload |
| --- | --- | --- | --- | --- | --- |
| .NET | `otel-dotnet-validation` | `dotnet-operator-validation.yaml` | Included | `15010:5000` | `run-dotnet-product-workload.sh` |
| Go | `otel-go-validation` | `go-operator-validation.yaml` | `go-ddot-collector.yaml` | `15003:5000` | `run-go-product-scenario.sh` |
| Java | `otel-java-validation` | `java-operator-validation.yaml` | `java-ddot-collector.yaml` | `15001:5000` | `run-java-product-workload.sh` |
| Node.js | `otel-node-validation` | `node-operator-validation.yaml` | `node-ddot-collector.yaml` | `15004:5000` | `run-node-product-workload.sh` |
| Python | `otel-python-validation` | `python-operator-validation.yaml` | `ddot-collector.yaml` | `15000:5000` | `run-python-product-workload.sh` |

The manifests pin the SDK and Agent versions used for the original validation.
Build each local image with the tag referenced by its deployment before
applying the manifest.

For focused Python feature checks, run
`scripts/run-python-feature-scenario.sh <scenario> http://127.0.0.1:15000`.
Use `dbm-full` with the committed deployment. For `dbm-service`, set
`DD_DBM_PROPAGATION_MODE=service`, run the scenario, and restore the manifest
afterward.

Switch a deployed lane between standalone DDOT and Datadog native trace export
with `scripts/set-trace-export.sh`. This leaves the application image, service
identity, Agent, database, and workload unchanged.
