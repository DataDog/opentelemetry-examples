from flask import Flask, request, jsonify
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode
import time
import random

from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
import logging

app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)
RequestsInstrumentor().instrument()

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

tracer = trace.get_tracer(__name__)

scores = {}


@app.route("/update_score", methods=["POST"])
def update_score():
    with tracer.start_as_current_span("update_score") as span:
        ms = max(0, random.gauss(1750, 1000))
        time.sleep(ms / 1000.0)
        try:
            player = request.json.get("player")
            result = request.json.get("result")

            # Introduce an error condition
            if not player or not result:
                raise ValueError("Player or result is missing")

            score = scores.get(player, 0) + int(result)
            scores[player] = score
            return jsonify({"player": player, "score": score})

        except ValueError as e:
            logger.warning(f"An error occurred: {e}")
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            return jsonify({"An error occurred", "Oops!"}), 400


if __name__ == "__main__":
    app.run(port=5001)
