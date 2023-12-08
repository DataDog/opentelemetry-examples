# OpenTelemetry Examples

This repository includes example applications and configurations for Datadog users, engineers, and support to understand how Datadog support of [OpenTelemetry][1] works today. These examples are intended to serve as references for learning how to integrate OpenTelemetry instrumented applications with Datadog products, and can be run independently to experiment with OpenTelemetry behaviors. Available resources are:
- [Common mistakes][14] for common mistakes or confusions end-users face when using OpenTelemetry.
- [Configurations][15] for configuration examples of the Datadog Agent and the OpenTelemetry Collector.

## Index

| Example  | Use Cases |
| ------------- | ------------- |
| [Game of Life][2]  | Go client-server app using gRPC, OTel Go and dd-trace-go instrumentations  |
| [Kafka Redis Messages][3] | Distributed app with Kafka and Redis, OTel and Datadog instrumentations  |
| [Log Trace Correlation][4] | Go client-server app demonstrating log trace correlation in Datadog, OTel instrumentation with Datadog Agent |
| [Manual Container Metrics][5] | Go server demonstrating container metrics correlating with traces in the Datadog trace app, OTel Go instrumentation with OTel Collector  |
| .[NET REST Service][6] | .NET API app, OTel .NET and dd-trace-dotnet instrumentations  |
| [Go REST Service][7] | Go HTTP app, OTel Go instrumentation  |
| [Java REST Service][8] | Java HTTP app, OTel Java and dd-trace-java instrumentation, probabilistic sampler processor |
| [JavaScript REST Service][9] | Node.js API app, OTel Node.js auto instrumentation |
| [Python REST Service][10] | CRUD API apps for both SQLite & Postgres, OTel Python auto instrumentation  |
| [RPC][11] | Go client-server gRPC app, OpenCensus gRPC plug-in, OpenCensus Bridge from OpenTelemetry, OTLP exporter  |
| [Span Links][12] | Distributed app with OTel span links, OTel Go and Java instrumentations |
| [W3C Trace Context][13] | Java and Python app to demonstrate W3C trace context propagation between OTel and DD instrumented apps |


[1]: https://opentelemetry.io/
[2]: ./apps/game-of-life/
[3]: ./apps/kafka-redis-messages/
[4]: ./apps/log-trace-correlation/
[5]: ./apps/manual-container-metrics/
[6]: ./apps/rest-services/dotnet/
[7]: ./apps/rest-services/golang/
[8]: ./apps/rest-services/java/
[9]: ./apps/rest-services/js/
[10]: ./apps/rest-services/py/
[11]: ./apps/rpc/
[12]: ./apps/span-links/
[13]: ./apps/w3-trace-context/
[14]: ./guides/common-mistakes.md
[15]: ./configurations/