# Manual instrumentation for exponential histograms
import os
import time
import random
import logging
from flask import Flask, request, jsonify

# OpenTelemetry setup with exponential histogram views
from opentelemetry import trace, metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.view import View
from opentelemetry.sdk.metrics._internal.aggregation import ExponentialBucketHistogramAggregation
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.resources import Resource
from opentelemetry.trace import Status, StatusCode

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
    "service.name": "scorey",
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
    export_interval_millis=5000  # Export every 5 seconds
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
tracer = trace.get_tracer("manual-scoring-tracer")
meter = metrics.get_meter("manual-scoring-meter")

# Create HTTP duration histogram with exponential aggregation
http_duration_histogram = meter.create_histogram(
    "http.server.request.duration",
    description="HTTP server request duration",
    unit="s"
)

# Create counter for score tracking
scores_updated_counter = meter.create_counter(
    "scores.updated",
    description="Number of score updates",
)

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

scores = {}

@app.route("/update_score", methods=["POST"])
def update_score():
    start_time = time.time()
    
    with tracer.start_as_current_span("update_score") as span:
        # Gaussian sleep
        ms = max(0, random.gauss(1750, 1000))
        time.sleep(ms / 1000.0)
        
        try:
            player = request.json.get("player")
            result = request.json.get("result")

            if not player or not result:
                raise ValueError("Player or result is missing")

            score = scores.get(player, 0) + int(result)
            scores[player] = score
            
            span.set_attribute("player", player)
            span.set_attribute("result", result)
            span.set_attribute("new_score", score)
            
            # Record metrics manually
            end_time = time.time()
            duration = end_time - start_time
            
            # Record HTTP duration with proper attributes
            http_duration_histogram.record(duration, {
                "http.request.method": "POST",
                "http.response.status_code": 200,
                "url.path": "/update_score",
                "http.route": "/update_score"
            })
            
            # Count the score update
            scores_updated_counter.add(1, {"player": player})
            
            return jsonify({"player": player, "score": score})

        except ValueError as e:
            logger.warning(f"An error occurred: {e}")
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            
            # Record error metric
            end_time = time.time()
            duration = end_time - start_time
            http_duration_histogram.record(duration, {
                "http.request.method": "POST",
                "http.response.status_code": 400,
                "url.path": "/update_score",
                "http.route": "/update_score"
            })
            
            return jsonify({"error": "An error occurred"}), 400

@app.route("/health", methods=["GET"])
def health():
    # No metrics for health to keep them clean
    return jsonify({"status": "healthy"}), 200

if __name__ == "__main__":
    print(f"🎯 Starting manual instrumentation with {len(views)} views")
    app.run(debug=False, port=5001, host='0.0.0.0')