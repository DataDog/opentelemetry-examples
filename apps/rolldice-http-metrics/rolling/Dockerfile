FROM python:3.9-alpine

WORKDIR /rolling

COPY . /rolling

RUN pip install -r requirements.txt

EXPOSE 5004

ENV FLASK_RUN_PORT=5004
ENV FLASK_RUN_HOST=0.0.0.0

CMD ["opentelemetry-instrument", "--service_name", "dicey", "--logs_exporter", "otlp", "flask", "run"]
