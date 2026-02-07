const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');

// Initialize the OTel SDK before any other imports/requires.
// The NodeSDK automatically picks up OTEL_SERVICE_NAME, OTEL_EXPORTER_OTLP_ENDPOINT,
// and OTEL_EXPORTER_OTLP_PROTOCOL from environment variables.
// W3C TraceContext propagation is the default propagator.
const sdk = new NodeSDK({
  instrumentations: [
    getNodeAutoInstrumentations({
      // Express and HTTP instrumentations are enabled by default.
      // This ensures automatic context propagation across HTTP calls.
      '@opentelemetry/instrumentation-fs': {
        enabled: false, // Disable filesystem instrumentation to reduce noise
      },
    }),
  ],
});

sdk.start();

const express = require('express');
const axios = require('axios');
const { context, trace, propagation, SpanStatusCode } = require('@opentelemetry/api');

const app = express();
app.use(express.json());

// Health check endpoint for Docker health checks and readiness probes
app.get('/health', (_req, res) => {
  res.status(200).json({ status: 'ok', service: 'game_controller' });
});

app.post('/play_game', async (req, res) => {
  const tracer = trace.getTracer('game-controller');
  const span = tracer.startSpan('play_game');
  const ctx = trace.setSpan(context.active(), span);

  await context.with(ctx, async () => {
    try {
      const player = req.body.player;

      if (!player) {
        span.setStatus({ code: SpanStatusCode.ERROR, message: 'Player name is required' });
        span.end();
        return res.status(400).json({ error: 'Player name is required' });
      }

      // W3C trace context headers are automatically injected by the HTTP instrumentation.
      // The auto-instrumentation for axios/http handles context propagation,
      // so manual propagation.inject() is not needed here.
      const diceRollResult = await axios.get(
        `http://rolling:5004/rolldice?player=${encodeURIComponent(player)}`
      );

      const updateScoreResult = await axios.post('http://scoring:5001/update_score', {
        player: player,
        result: diceRollResult.data,
      });

      span.setAttribute('game.player', player);
      span.setAttribute('game.roll_result', String(diceRollResult.data));
      span.addEvent('score_updated');
      span.setStatus({ code: SpanStatusCode.OK });
      span.end();
      res.json(updateScoreResult.data);
    } catch (error) {
      span.recordException(error);
      span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
      span.end();
      console.error('Error in play_game:', error.message);
      res.status(500).json({ error: 'An error occurred while processing the game' });
    }
  });
});

const PORT = process.env.PORT || 5002;

const server = app.listen(PORT, () => {
  console.log(`Game Controller service listening at http://localhost:${PORT}`);
});

// Graceful shutdown: flush OTel data and close the server
function shutdown(signal) {
  console.log(`Received ${signal}. Shutting down gracefully...`);
  server.close(() => {
    sdk.shutdown()
      .then(() => {
        console.log('OTel SDK shut down successfully.');
        process.exit(0);
      })
      .catch((err) => {
        console.error('Error shutting down OTel SDK:', err);
        process.exit(1);
      });
  });
}

process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT', () => shutdown('SIGINT'));
