# .NET OTel Operator product validation

This ASP.NET Core app pins `Datadog.Trace.Bundle` 3.53.0 and Npgsql 8.0.6. The
Datadog tracer exports traces to standalone DDOT as OTLP HTTP/JSON. Profiling,
Dynamic Instrumentation, and DBM correlation use their normal Agent side
channels.

The workload covers an ActivitySource bridge span, four local Dynamic
Instrumentation probes, CPU/allocation/exception profiles, and three
deterministic PostgreSQL query shapes. The Kubernetes deployment lives in
`k8s/dotnet-operator-validation.yaml`; `scripts/run-dotnet-product-workload.sh`
drives a complete test window.
