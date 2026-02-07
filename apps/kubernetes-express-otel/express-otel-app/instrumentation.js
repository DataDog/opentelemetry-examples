// instrumentation.js
// References:
//   https://opentelemetry.io/docs/languages/js/getting-started/nodejs/
//   https://opentelemetry.io/docs/languages/js/exporters/
const opentelemetry = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-proto');
const { OTLPMetricExporter } = require('@opentelemetry/exporter-metrics-otlp-proto');
const { PeriodicExportingMetricReader } = require('@opentelemetry/sdk-metrics');
const { diag, DiagConsoleLogger, DiagLogLevel } = require('@opentelemetry/api');

// Set diagnostic log level from environment variable or default to INFO.
// Use OTEL_LOG_LEVEL=debug for troubleshooting.
const logLevel = (process.env.OTEL_LOG_LEVEL || 'info').toUpperCase();
const diagLogLevel = DiagLogLevel[logLevel] != null ? DiagLogLevel[logLevel] : DiagLogLevel.INFO;
diag.setLogger(new DiagConsoleLogger(), diagLogLevel);

const sdk = new opentelemetry.NodeSDK({
  traceExporter: new OTLPTraceExporter(),
  metricReader: new PeriodicExportingMetricReader({
    exporter: new OTLPMetricExporter(),
  }),
  instrumentations: [
    getNodeAutoInstrumentations({
      // Disable fs instrumentation to reduce noise
      '@opentelemetry/instrumentation-fs': { enabled: false },
    }),
  ],
});

sdk.start();
