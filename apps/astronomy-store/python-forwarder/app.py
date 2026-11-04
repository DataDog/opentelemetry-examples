import json
import logging
import os
import sys

from confluent_kafka import Consumer, Producer
from opentelemetry import propagate, trace
from opentelemetry.trace import SpanKind

logging.basicConfig(stream=sys.stdout, level=logging.INFO, format="%(message)s")
logger = logging.getLogger("python-forwarder")

KAFKA_ADDR = os.environ.get("KAFKA_ADDR", "kafka:9092")
KAFKA_TOPIC = os.environ.get("KAFKA_TOPIC", "go-producer-orders")
KAFKA_OUTPUT_TOPIC = os.environ.get("KAFKA_OUTPUT_TOPIC", "python-forwarder-orders")
KAFKA_CONSUMER_GROUP = "python-forwarder"

tracer = trace.get_tracer(__name__)


def log(msg: str, **fields) -> None:
    logger.info(json.dumps({"level": "info", "msg": msg, **fields}))


def main() -> None:
    consumer = Consumer(
        {
            "bootstrap.servers": KAFKA_ADDR,
            "group.id": KAFKA_CONSUMER_GROUP,
            "auto.offset.reset": "earliest",
        }
    )
    producer = Producer({"bootstrap.servers": KAFKA_ADDR})
    consumer.subscribe([KAFKA_TOPIC])
    log("python-forwarder started", input_topic=KAFKA_TOPIC, output_topic=KAFKA_OUTPUT_TOPIC)
    try:
        while True:
            message = consumer.poll(timeout=1.0)
            if message is None:
                continue
            if message.error():
                log("kafka consumer error", error=str(message.error()))
                continue
            carrier = {key: value.decode("utf-8") for key, value in message.headers() or []}
            with tracer.start_as_current_span("forward_order", context=propagate.extract(carrier), kind=SpanKind.CONSUMER):
                log("forward_order", key=(message.key() or b"").decode("utf-8"))
                producer.produce(KAFKA_OUTPUT_TOPIC, key=message.key(), value=message.value())
                producer.poll(0)
    finally:
        producer.flush(10)
        consumer.close()


if __name__ == "__main__":
    main()
