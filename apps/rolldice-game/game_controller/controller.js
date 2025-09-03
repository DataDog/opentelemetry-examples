const express = require('express');
const axios = require('axios');
const { context, trace, propagation, metrics } = require('@opentelemetry/api');
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { HttpInstrumentation } = require('@opentelemetry/instrumentation-http');
const { ExpressInstrumentation } = require('@opentelemetry/instrumentation-express');

// Create HTTP instrumentation with metrics enabled
const httpInstrumentation = new HttpInstrumentation({
  // Enable metrics collection
  createSpanOnRequest: true,
  // Request hook to capture request metrics
  requestHook: (span, request) => {
    // Metrics will be automatically captured by the instrumentation
  },
  // Response hook to capture response metrics
  responseHook: (span, response) => {
    // Metrics will be automatically captured by the instrumentation
  }
});

const sdk = new NodeSDK({
  instrumentations: [
    getNodeAutoInstrumentations({
      '@opentelemetry/instrumentation-http': false, // Disable auto HTTP to use custom one
    }),
    httpInstrumentation,
    new ExpressInstrumentation(),
  ],
});

sdk.start();
const app = express();
app.use(express.json());

app.post('/play_game', async (req, res) => {
  const tracer = trace.getTracer('express-game-controller');
  const span = tracer.startSpan('play_game');
  
  const ctx = trace.setSpan(context.active(), span);

  context.with(ctx, async () => {
    try {
      const player = req.body.player;
      const headers = {};
      propagation.inject(context.active(), headers);
      const diceRollResult = await axios.get(`http://rolling:5004/rolldice?player=${player}`, { headers });
      const updateScoreResult = await axios.post('http://scoring:5001/update_score', {
        player: player,
        result: diceRollResult.data
      }, { headers });

      span.addEvent('score_updated');
      span.end();
      res.json(updateScoreResult.data);
    } catch (error) {
      span.recordException(error);
      span.end();
      console.error(error);
      res.status(500).send('An error occurred');
    }
  });
});

const PORT = process.env.PORT || 5002;
app.listen(PORT, () => {
  console.log(`Game Controller service listening at http://localhost:${PORT}`);
});
