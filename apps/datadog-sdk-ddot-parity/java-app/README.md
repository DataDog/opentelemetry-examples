# Java OTel Operator product validation app

This JDK 21 sample pins `dd-java-agent` 1.66.0 and exports traces as OTLP
HTTP/protobuf to standalone DDOT. It uses the Datadog Agent separately for
profiling, Dynamic Instrumentation, and PostgreSQL Database Monitoring.

The endpoints generate deterministic OTel API spans, OTel HTTP and database
semantic attributes, CPU work, allocations, exceptions, and JDBC queries.
`probes.json` supplies repeatable metric, conditional, span, span-tag, and log
probes without requiring probes to be created manually in the UI.
