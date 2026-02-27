# Unless explicitly stated otherwise all files in this repository are dual-licensed
# under the Apache 2.0 or BSD3 Licenses.
#
# This product includes software developed at Datadog (https://www.datadoghq.com/)
# Copyright 2022 Datadog, Inc.
import logging
import os
import signal
import sys
from datetime import datetime, timedelta
from random import randint

from flask import Flask, jsonify, request

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
    stream=sys.stdout,
)
logger = logging.getLogger(__name__)

app = Flask(__name__)


@app.route("/")
def index():
    return "Welcome to the calendar app!"


@app.route("/health")
def health():
    """Health check endpoint for container orchestration."""
    return jsonify({"status": "healthy"}), 200


@app.route("/server_request")
def server_request():
    return "served"


@app.route("/calendar")
def get_date():
    """Generates a random date in 2022.

    The traceparent and tracestate headers are automatically propagated
    by the instrumentation agent (OTel or ddtrace) when this service
    is called from an upstream service. No manual header handling is
    required -- the agent instruments Flask and the requests library
    to inject/extract W3C Trace Context headers transparently.
    """
    try:
        day_offset = randint(0, 365)
        start_date = datetime(2022, 1, 1)
        output = start_date + timedelta(days=day_offset)
        date_str = output.strftime("%m/%d/%Y")
        logger.info("Generated date: %s", date_str)
        return date_str
    except Exception:
        logger.exception("Error generating date")
        return jsonify({"error": "internal server error"}), 500


def _handle_shutdown(signum, frame):
    """Handle SIGTERM/SIGINT for graceful shutdown."""
    logger.info("Received signal %s, shutting down gracefully", signum)
    sys.exit(0)


if __name__ == "__main__":
    signal.signal(signal.SIGTERM, _handle_shutdown)
    signal.signal(signal.SIGINT, _handle_shutdown)

    port = int(os.environ.get("SERVER_PORT", 9090))
    logger.info("Starting calendar service on port %d", port)
    app.run(host="0.0.0.0", port=port)
