import logging
import random
import sys

import structlog
from flask import Flask, jsonify
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
app = Flask(__name__)


@app.post("/fraud-detection/check-order")
def check_order():
    fraud_score = random.randrange(100)
    trace.get_current_span().set_attribute("astronomystore.fraud_score", fraud_score)
    logger.info("check_order", fraud_score=fraud_score)
    return jsonify(fraud_score=fraud_score), 200


@app.get("/health")
def health():
    return "", 200
