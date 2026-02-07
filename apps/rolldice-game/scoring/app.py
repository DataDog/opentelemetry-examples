import logging
import signal
import sys

from flask import Flask, request, jsonify
from opentelemetry import trace
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.logging import LoggingInstrumentor
from opentelemetry.trace import Status, StatusCode

# Configure structured logging with OTel trace correlation fields.
# The LoggingInstrumentor injects trace_id and span_id into log records,
# enabling log-trace correlation in Datadog.
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s [%(name)s] [trace_id=%(otelTraceID)s span_id=%(otelSpanID)s] %(message)s",
)
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Instrument Flask to auto-create spans for incoming HTTP requests.
# This also extracts W3C traceparent headers from incoming requests
# so that the scoring service's spans are children of the controller's trace.
FlaskInstrumentor().instrument_app(app)
LoggingInstrumentor().instrument(set_logging_format=True)

tracer = trace.get_tracer(__name__)

# In-memory score storage (for demo purposes)
scores = {}


@app.route("/health")
def health():
    """Health check endpoint for Docker health checks and readiness probes."""
    return {"status": "ok", "service": "scoring"}, 200


@app.route("/update_score", methods=["POST"])
def update_score():
    with tracer.start_as_current_span("update_score") as span:
        try:
            data = request.json
            if data is None:
                raise ValueError("Request body must be JSON")

            player = data.get("player")
            result = data.get("result")

            if not player or not result:
                raise ValueError("Player or result is missing")

            score = scores.get(player, 0) + int(result)
            scores[player] = score

            # Set semantic attributes that map to Datadog span tags.
            span.set_attribute("game.player", player)
            span.set_attribute("game.score", score)
            span.set_attribute("game.roll_result", str(result))
            span.set_status(Status(StatusCode.OK))

            logger.info("Score updated for %s: %d", player, score)
            return jsonify({"player": player, "score": score})

        except (ValueError, TypeError) as e:
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            logger.error("Error updating score: %s", str(e))
            return jsonify({"error": str(e)}), 400

        except Exception as e:
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            logger.error("Unexpected error updating score: %s", str(e))
            return jsonify({"error": "Internal server error"}), 500


def graceful_shutdown(signum, frame):
    """Handle shutdown signals to ensure OTel data is flushed."""
    logger.info("Received signal %s, shutting down...", signum)
    sys.exit(0)


signal.signal(signal.SIGTERM, graceful_shutdown)
signal.signal(signal.SIGINT, graceful_shutdown)


if __name__ == "__main__":
    app.run(debug=False, host="0.0.0.0", port=5001)
