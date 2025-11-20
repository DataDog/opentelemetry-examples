# Score Server

This is a simple Flask application that tracks scores for players. It uses the OpenTelemetry API for tracing.

## Installation
* Python (v3.9.16)
* Use the package manager [pip](https://pip.pypa.io/en/stable/) to install `requirements.txt`.

```bash
pip install requirements.txt
```

## Run the server

```bash
opentelemetry-instrument --service_name scorey --logs_exporter otlp flask run -p 5001
```
