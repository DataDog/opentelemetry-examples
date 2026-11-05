import pino from "pino";

// JSON logs on stdout. pino is in Next.js' default `serverExternalPackages`, so it's loaded
// with `require` rather than bundled, letting the auto-instrumentation patch it to add trace context.
// `base` drops pino's default `hostname` field, which would be the pod name rather than the node's host name.
export const logger = pino({
  level: process.env.LOG_LEVEL ?? "info",
  base: { pid: process.pid },
});
