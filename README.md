# OpenTelemetry Examples

The repository includes example applications and configurations for Datadog users, engineers, and support to understand how Datadog support of [OpenTelemetry][1] works today. These examples provide reference material for integrating OpenTelemetry instrumented applications with Datadog products and allow independent experimentation with OpenTelemetry behaviors. The available resources include:
- [Common mistakes][14] for common mistakes or confusions end-users face when using OpenTelemetry.
- [Configurations][15] for configuration examples of the Datadog Agent and the OpenTelemetry Collector.

## Index

| Example  | Description | Monitoring Capabilities |
| ------------- | ------------- | ------------- |
| [Game of Life][2]  | Go client-server app using gRPC, OTel Go and dd-trace-go instrumentations | Tracing gRPC and HTTP endpoints, OTLP metrics and logs, runtime metrics |
| [Kafka Redis Messages][3] | Distributed app with Kafka and Redis, OTel and Datadog instrumentations | Kafka, Redis |
| [Log Trace Correlation][4] | Go client-server app automatically generating traces and logs, OTel instrumentation with Datadog Agent | Log trace correlation |
| [Manual Container Metrics][5] | Go server manually generating OTel container metrics, OTel Go instrumentation with OTel Collector | Container metrics correlation in the trace app |
| .[NET REST Service][6] | .NET API app, OTel .NET and dd-trace-dotnet instrumentations | Tracing, runtime metrics |
| [Go REST Service][7] | Go HTTP app, OTel Go instrumentation | Tracing HTTP endpoint, OTel metrics, runtime metrics |
| [Java REST Service][8] | Java HTTP app, OTel Java and dd-trace-java instrumentation, probabilistic sampler processor | Ingestion sampling, runtime metrics |
| [JavaScript REST Service][9] | Node.js API app, OTel Node.js auto instrumentation | Auto instrumented traces |
| [Python REST Service][10] | CRUD API apps, OTel Python auto instrumentation | SQLite & Postgres |
| [RPC][11] | Go client-server gRPC Hello World app | Tracing gRPC endpoint, gRPC metrics using OpenCensus bridge |
| [Span Links][12] | Distributed app with Kafka messages, OTel Go and Java instrumentations | OTel span links |
| [W3C Trace Context][13] | Java and Python app to demonstrate W3C trace context propagation between OTel and DD instrumented apps | W3C trace context, runtime metrics |
| [Kubernetes (Datadog Operator and Helm) with Express][15] | An Express sample app configured with Kubernetes  | Kubernetes |
| [Python and Javascript trace context propagation][17] | An Express controller server calling two Flask servers | Standalone Host |
| [Kafka Producer, Consumer and Broker][18] | A kafka java consumer, java producer and broker | Kafka metrics, Tracing, Logs |

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
[13]: ./apps/w3c-trace-context/
[14]: ./guides/common-mistakes.md
[15]: ./configurations/
[16]: ./apps/kubernetes-express-otel/
[17]: ./apps/w3c-trace-context-ex2/
[18]: ./apps/kafka-metrics/