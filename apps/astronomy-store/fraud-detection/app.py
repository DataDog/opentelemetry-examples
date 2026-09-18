import json
import logging
import os
import random
import sys

import structlog
from confluent_kafka import Consumer
from opentelemetry import trace


def configure_logging() -> None:
    formatter = structlog.stdlib.ProcessorFormatter(
        processor=structlog.processors.JSONRenderer(),
        foreign_pre_chain=[
            structlog.stdlib.add_log_level,
            structlog.processors.TimeStamper(fmt="iso", utc=True),
        ],
    )
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(formatter)
    root_logger = logging.getLogger()
    root_logger.handlers.clear()
    root_logger.addHandler(handler)
    root_logger.setLevel(logging.INFO)

    structlog.configure(
        processors=[
            structlog.stdlib.filter_by_level,
            structlog.stdlib.add_logger_name,
            structlog.stdlib.add_log_level,
            structlog.stdlib.ExtraAdder(),
            structlog.processors.TimeStamper(fmt="iso", utc=True),
            structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
        ],
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )


configure_logging()
logger = structlog.get_logger(__name__)

KAFKA_ADDR = os.environ.get("KAFKA_ADDR", "kafka:9092")
KAFKA_TOPIC = os.environ.get("KAFKA_TOPIC", "orders")
KAFKA_CONSUMER_GROUP = "fraud-detection"


def check_order(order: dict) -> int:
    fraud_score = random.randrange(100)
    trace.get_current_span().set_attribute("astronomystore.fraud_score", fraud_score)
    logger.info(
        "check_order",
        order_id=order.get("order_id"),
        fraud_score=fraud_score,
    )
    return fraud_score


def main() -> None:
    consumer = Consumer(
        {
            "bootstrap.servers": KAFKA_ADDR,
            "group.id": KAFKA_CONSUMER_GROUP,
            "auto.offset.reset": "earliest",
        }
    )
    consumer.subscribe([KAFKA_TOPIC])
    logger.info(
        "fraud-detection consumer started",
        bootstrap_servers=KAFKA_ADDR,
        topic=KAFKA_TOPIC,
        group_id=KAFKA_CONSUMER_GROUP,
    )
    try:
        while True:
            message = consumer.poll(timeout=1.0)
            if message is None:
                continue
            if message.error():
                logger.error("kafka consumer error", error=str(message.error()))
                continue
            check_order(json.loads(message.value().decode("utf-8")))
    finally:
        consumer.close()


if __name__ == "__main__":
    main()
