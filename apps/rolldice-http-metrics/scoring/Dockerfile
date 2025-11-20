FROM python:3.9-alpine

WORKDIR /scoring

COPY . /scoring

RUN pip install -r requirements.txt

EXPOSE 5001

ENV FLASK_RUN_PORT=5001
ENV FLASK_RUN_HOST=0.0.0.0

CMD ["opentelemetry-instrument", "--service_name", "scorey", "--logs_exporter", "otlp", "flask", "run"]
