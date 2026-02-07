FROM python:3.12-slim

WORKDIR /app

# Copy and install dependencies first for better Docker layer caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application source
COPY app.py .

EXPOSE 5001

ENV FLASK_RUN_PORT=5001
ENV FLASK_RUN_HOST=0.0.0.0

# Health check to verify the service is responding
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:5001/health')" || exit 1

# Use opentelemetry-instrument to auto-configure the OTel SDK and export telemetry.
# The --service_name here is a fallback; OTEL_SERVICE_NAME env var takes precedence.
# --logs_exporter otlp enables sending logs via OTLP for log-trace correlation in Datadog.
CMD ["opentelemetry-instrument", "--service_name", "scoring", "--logs_exporter", "otlp", "flask", "run"]
