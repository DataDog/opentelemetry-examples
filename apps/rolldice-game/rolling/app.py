import logging
import signal
import sys
from random import randint

from flask import Flask, request
from opentelemetry import metrics, trace
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.logging import LoggingInstrumentor
from opentelemetry.trace import StatusCode

# Configure structured logging.
# The LoggingInstrumentor injects trace_id, span_id, and service name into log records,
# which enables log-trace correlation in Datadog.
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s [%(name)s] [trace_id=%(otelTraceID)s span_id=%(otelSpanID)s] %(message)s",
)
logger = logging.getLogger(__name__)

# Initialize tracer and meter.
# When using opentelemetry-instrument CLI, the SDK is auto-configured from env vars:
#   OTEL_SERVICE_NAME, OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_PROTOCOL
# W3C TraceContext propagation is the default, ensuring distributed trace linking.
tracer = trace.get_tracer("diceroller.tracer")
meter = metrics.get_meter("diceroller.meter")

# Create a counter instrument for tracking dice rolls.
# This metric appears in Datadog as a custom metric with the roll.value dimension.
roll_counter = meter.create_counter(
    "dice.rolls",
    description="The number of rolls by roll value",
)

app = Flask(__name__)

# Instrument Flask to auto-create spans for incoming HTTP requests.
# This also extracts W3C traceparent headers from incoming requests
# so that the rolling service's spans are children of the controller's trace.
FlaskInstrumentor().instrument_app(app)
LoggingInstrumentor().instrument(set_logging_format=True)


@app.route("/health")
def health():
    """Health check endpoint for Docker health checks and readiness probes."""
    return {"status": "ok", "service": "rolling"}, 200


@app.route("/rolldice")
def roll_dice():
    with tracer.start_as_current_span("roll_dice_operation") as roll_dice_span:
        player = request.args.get("player", default=None, type=str)

        with tracer.start_as_current_span("generate_roll_result"):
            result = str(roll())
            # Set semantic attributes that map to Datadog span tags.
            roll_dice_span.set_attribute("roll.value", result)
            roll_dice_span.set_attribute("game.player", player or "anonymous")

        # Count the roll with the value as a dimension
        roll_counter.add(1, {"roll.value": result})

        with tracer.start_as_current_span("log_roll_result"):
            if player:
                logger.info("%s is rolling the dice: %s", player, result)
            else:
                logger.info("Anonymous player is rolling the dice: %s", result)

        roll_dice_span.set_status(StatusCode.OK)
        return result


def roll():
    return randint(1, 6)


def graceful_shutdown(signum, frame):
    """Handle shutdown signals to ensure OTel data is flushed."""
    logger.info("Received signal %s, shutting down...", signum)
    sys.exit(0)


signal.signal(signal.SIGTERM, graceful_shutdown)
signal.signal(signal.SIGINT, graceful_shutdown)


if __name__ == "__main__":
    app.run(debug=False, host="0.0.0.0", port=5004)
