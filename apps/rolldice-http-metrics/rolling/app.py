# CRITICAL: Set up exponential histograms BEFORE any auto-instrumentation imports
import os

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

# NOW import auto-instrumentation (after meter provider is locked in)
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.logging import LoggingInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor

from opentelemetry import trace
from random import randint
from flask import Flask, request
import logging
import random
import time

# Initialize tracer and meter
tracer = trace.get_tracer("diceroller.tracer")
meter = metrics.get_meter("diceroller.meter")

# Create a counter instrument
roll_counter = meter.create_counter(
    "dice.rolls",
    description="The number of rolls by roll value",
)

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@app.route("/rolldice")
def roll_dice():
    with tracer.start_as_current_span("roll_dice_operation") as roll_dice_span:
        ms = max(0, random.gauss(1750, 1000))
        time.sleep(ms / 1000.0)
        player = request.args.get("player", default=None, type=str)

        # Introducing an additional span for the "roll" function
        with tracer.start_as_current_span("generate_roll_result"):
            result = str(roll())
            roll_dice_span.set_attribute("roll.value", result)

        # Count the roll
        roll_counter.add(1, {"roll.value": result})

        # Additional span for logging the result
        with tracer.start_as_current_span("log_roll_result"):
            if player:
                logger.warning(f"{player} is rolling the dice: {result}")
            else:
                logger.warning(f"Anonymous player is rolling the dice: {result}")
        return result


def roll():
    # You can also add a span inside this function if it were more complex
    return randint(1, 6)


@app.route("/health", methods=["GET"])
def health():
    return "healthy", 200


if __name__ == "__main__":
    # Instrumenting Flask, Logging, and Requests
    FlaskInstrumentor().instrument_app(app)
    LoggingInstrumentor().instrument(set_logging_format=True)
    RequestsInstrumentor().instrument()
    app.run(debug=False, port=5004)