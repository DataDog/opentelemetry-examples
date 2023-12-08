# Distributed Calendar Application

A simple distributed application with Kafka and Redis. We have instrumented the same set of services with OpenTelemetry & DD libraries.

![image](https://user-images.githubusercontent.com/2471669/230177263-e65a2b05-1d83-482d-93f2-2e56ee45fa25.png)

## dd
This application is instrumented with Datadog language libraries.

## otel
This application is instrumented with OTel SDKs.

## otel-api-with-dd
This application demonstrates OTel API compatability with Datadog language libraries. The Java Calendar app uses OTel APIs and instrumented with dd-trace-java. The Go Calendar app uses OTel APIs and is instrumented with OpenTelemetry Java.
