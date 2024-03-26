# Dice Server

This is a simple Flask application that simulates a dice roll. It uses the OpenTelemetry API for tracing and metrics, and Python's built-in logging module for logging.

## Installation
* Python (v3.9.16)
* Use the package manager pip to install `requirements.txt`.

```bash
pip install requirements.txt
```

## Run the server

```bash
opentelemetry-instrument --service_name dicey --logs_exporter otlp flask run -p 8080```

