//instrumentation.js from 
//    https://opentelemetry.io/docs/languages/js/getting-started/nodejs/*/
//    https://opentelemetry.io/docs/languages/js/exporters/
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-proto');
const { diag, DiagConsoleLogger, DiagLogLevel } = require('@opentelemetry/api');

// Enable debug mode for the SDK
diag.setLogger(new DiagConsoleLogger(), DiagLogLevel.DEBUG);

console.log("The OTLP endpoint")
console.log(process.env.OTEL_EXPORTER_OTLP_ENDPOINT)
const sdk = new NodeSDK({
  traceExporter: new OTLPTraceExporter({url: process.env.OTEL_EXPORTER_OTLP_ENDPOINT}),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();
