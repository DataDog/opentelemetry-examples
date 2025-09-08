# CRITICAL: Set up exponential histograms BEFORE any auto-instrumentation imports  
import os
import time
import random

# Only import OpenTelemetry core, not instrumentation yet
from opentelemetry import metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.view import View
from opentelemetry.sdk.metrics._internal.aggregation import ExponentialBucketHistogramAggregation

# Configure exponential histogram views FIRST, before any instrumentation imports
views = []
if os.getenv('USE_EXPONENTIAL_HISTOGRAMS'):
    print("🔥 CONFIGURING EXPONENTIAL HISTOGRAMS 🔥")
    http_server_view = View(
        instrument_name="http.server.request.duration",
        aggregation=ExponentialBucketHistogramAggregation()
    )
    
    http_client_view = View(
        instrument_name="http.client.request.duration", 
        aggregation=ExponentialBucketHistogramAggregation()
    )
    
    views = [http_server_view, http_client_view]
    print(f"🔥 CREATED {len(views)} EXPONENTIAL VIEWS 🔥")
else:
    print("📊 Using standard histograms")

# Set meter provider with exponential views BEFORE any instrumentation
meter_provider = MeterProvider(views=views)
metrics.set_meter_provider(meter_provider)
print(f"🎯 METER PROVIDER SET WITH {len(views)} VIEWS 🎯")

# NOW import everything else (after meter provider is locked in)
from flask import Flask, request, jsonify
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
import logging

app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)
RequestsInstrumentor().instrument()

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

tracer = trace.get_tracer(__name__)
meter = metrics.get_meter(__name__)

# Create custom metric counter
scores_updated_counter = meter.create_counter(
    "scores.updated",
    description="Total number of score updates",
)

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
            
            # Increment custom metric
            scores_updated_counter.add(1, {"player": player})
            
            return jsonify({"player": player, "score": score})

        except ValueError as e:
            logger.warning(f"An error occurred: {e}")
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            return jsonify({"An error occurred", "Oops!"}), 400


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "healthy"}), 200


if __name__ == "__main__":
    app.run(port=5001)