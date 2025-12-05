# Manual instrumentation for exponential histograms
import os
import time
import random
import logging
from random import randint
from flask import Flask, request

# OpenTelemetry setup with exponential histogram views
from opentelemetry import trace, metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.view import View
from opentelemetry.sdk.metrics._internal.aggregation import ExponentialBucketHistogramAggregation
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.resources import Resource

# Configure exponential histogram views for HTTP metrics
views = []
if os.getenv('USE_EXPONENTIAL_HISTOGRAMS'):
    print("🔥 MANUAL EXPONENTIAL HISTOGRAMS ENABLED 🔥")
    http_server_view = View(
        instrument_name="http.server.request.duration",
        aggregation=ExponentialBucketHistogramAggregation()
    )
    
    http_client_view = View(
        instrument_name="http.client.request.duration", 
        aggregation=ExponentialBucketHistogramAggregation()
    )
    
    views = [http_server_view, http_client_view]
else:
    print("📊 Using standard histograms")

# Set up providers with resource attributes
deployment_env = "eks-demo-exponential" if os.getenv('USE_EXPONENTIAL_HISTOGRAMS') else "eks-demo"
resource = Resource.create({
    "service.name": "rolly",
    "service.version": "v1.0",
    "deployment.environment": deployment_env
})

# Configure OTLP exporter with proper export interval
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader

otlp_exporter = OTLPMetricExporter(
    endpoint=os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"),
    insecure=True
)

metric_reader = PeriodicExportingMetricReader(
    exporter=otlp_exporter,
    export_interval_millis=int(os.getenv("OTEL_METRIC_EXPORT_INTERVAL", "5000")),  # Export interval in milliseconds
    export_timeout_millis=int(os.getenv("OTEL_METRIC_EXPORT_TIMEOUT", "3000"))   # Export timeout in milliseconds
)

# Initialize meter provider with exponential views AND exporter
meter_provider = MeterProvider(
    views=views, 
    resource=resource,
    metric_readers=[metric_reader]
)
metrics.set_meter_provider(meter_provider)

# Initialize tracer provider
tracer_provider = TracerProvider(resource=resource)
trace.set_tracer_provider(tracer_provider)

# Get tracer and meter
tracer = trace.get_tracer("manual-rolling-tracer")
meter = metrics.get_meter("manual-rolling-meter")

# Create HTTP duration histogram with exponential aggregation
http_duration_histogram = meter.create_histogram(
    "http.server.request.duration",
    description="HTTP server request duration",
    unit="s"
)

# Create counter for roll tracking
roll_counter = meter.create_counter(
    "dice.rolls",
    description="Number of dice rolls",
)

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@app.route("/rolldice")
def roll_dice():
    start_time = time.time()
    
    with tracer.start_as_current_span("roll_dice_operation") as span:
        # Gaussian sleep
        ms = max(0, random.gauss(1750, 1000))
        time.sleep(ms / 1000.0)
        
        player = request.args.get("player", default=None, type=str)
        
        # Generate roll
        with tracer.start_as_current_span("generate_roll"):
            result = str(randint(1, 6))
            span.set_attribute("roll.value", result)
        
        # Log result
        if player:
            logger.warning(f"{player} is rolling the dice: {result}")
            span.set_attribute("player", player)
        else:
            logger.warning(f"Anonymous player is rolling the dice: {result}")
        
        # Record metrics manually
        end_time = time.time()
        duration = end_time - start_time
        
        # Record HTTP duration with proper attributes
        http_duration_histogram.record(duration, {
            "http.request.method": "GET",
            "http.response.status_code": 200,
            "url.path": "/rolldice",
            "http.route": "/rolldice"
        })
        
        # Count the roll
        roll_counter.add(1, {"roll.value": result})
        
        return result

@app.route("/health", methods=["GET"])
def health():
    # No metrics for health to keep them clean
    return "healthy", 200

if __name__ == "__main__":
    print(f"🎯 Starting manual instrumentation with {len(views)} views")
    app.run(debug=False, port=5004, host='0.0.0.0')