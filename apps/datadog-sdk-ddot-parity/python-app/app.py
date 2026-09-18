# isort: skip_file
import logging
import os
import asyncio
import threading
import time
from datetime import datetime, timezone

os.environ.setdefault("DD_TRACE_OTEL_ENABLED", "true")
os.environ.setdefault("DD_METRICS_OTEL_ENABLED", "true")
os.environ.setdefault("DD_LOGS_OTEL_ENABLED", "true")
os.environ.setdefault("OTEL_TRACES_EXPORTER", "otlp")

import ddtrace.auto  # noqa: E402,F401  must be imported before opentelemetry

from flask import Flask  # noqa: E402
from opentelemetry import metrics  # noqa: E402
from opentelemetry import trace  # noqa: E402
import psycopg  # noqa: E402
from probe_targets import (  # noqa: E402
    dynamic_probe_target,
    log_probe_target,
    metric_count_target,
    metric_distribution_target,
    metric_gauge_target,
    metric_histogram_target,
    snapshot_line_target,
    snapshot_method_target,
    span_decoration_target,
    span_probe_target,
)


app = Flask(__name__)
tracer = trace.get_tracer("otlp-python-demo")
meter = metrics.get_meter("otlp-python-demo")
requests_counter = meter.create_counter("demo.requests_total")
log = logging.getLogger("otlp-python-demo")
log.setLevel(logging.INFO)
allocation_hold = []
profile_lock = threading.Lock()


def database_connection():
    return psycopg.connect(
        host=os.getenv("POSTGRES_HOST", "postgres"),
        port=int(os.getenv("POSTGRES_DB_PORT", "5432")),
        dbname=os.getenv("POSTGRES_DB", "validation"),
        user=os.getenv("POSTGRES_USER", "validation"),
        password=os.getenv("POSTGRES_PASSWORD", "validation"),
        application_name=os.getenv("DD_SERVICE", "otel-python-validation"),
    )


@app.get("/health")
def health_endpoint():
    return {"status": "ok", "time": datetime.now(timezone.utc).isoformat()}


@app.get("/span")
def span_endpoint():
    time.sleep(0.5)
    with tracer.start_as_current_span("manual.span") as span:
        span.set_attribute("demo.endpoint", "/span")
    return {"span": "ok"}


@app.get("/metric")
def metric_endpoint():
    time.sleep(0.5)
    requests_counter.add(1, {"endpoint": "/metric"})
    return {"metric": "ok"}


@app.get("/log")
def log_endpoint():
    time.sleep(0.5)
    log.info("demo.log endpoint=/log")
    return {"log": "ok"}


@app.get("/success")
def success_endpoint():
    time.sleep(0.5)
    with tracer.start_as_current_span("demo.success") as span:
        span.set_attribute("demo.endpoint", "/success")
        requests_counter.add(1, {"endpoint": "/success", "outcome": "success"})
        log.info("demo.success endpoint=/success")
    return {"status": "ok"}


@app.get("/error")
def error_endpoint():
    time.sleep(0.5)
    with tracer.start_as_current_span("demo.error") as span:
        span.set_attribute("demo.endpoint", "/error")
        requests_counter.add(1, {"endpoint": "/error", "outcome": "error"})
        try:
            raise RuntimeError("demo.error: intentional failure for load-test verification")
        except RuntimeError as exc:
            span.record_exception(exc)
            span.set_status(trace.Status(trace.StatusCode.ERROR, str(exc)))
            log.error("demo.error endpoint=/error", exc_info=exc)
            return {"status": "error", "message": str(exc)}, 500


@app.get("/probe")
def probe_endpoint():
    from flask import request

    value = int(request.args.get("value", "7"))
    result = dynamic_probe_target(value)
    if request.args.get("error") == "1":
        return {"input": value, "result": result, "status": "intentional-error"}, 500
    return {"input": value, "result": result}


@app.get("/di/log")
def di_log_endpoint():
    value = int(request_arg("value", "7"))
    return {"input": value, "result": log_probe_target(value)}


@app.get("/di/snapshot-line")
def di_snapshot_line_endpoint():
    value = int(request_arg("value", "7"))
    return {"input": value, "result": snapshot_line_target(value)}


@app.get("/di/snapshot-method")
def di_snapshot_method_endpoint():
    value = int(request_arg("value", "7"))
    return {"input": value, "result": snapshot_method_target(value)}


@app.get("/di/metrics")
def di_metrics_endpoint():
    value = int(request_arg("value", "7"))
    return {
        "count": metric_count_target(value),
        "gauge": metric_gauge_target(value),
        "histogram": metric_histogram_target(value),
        "distribution": metric_distribution_target(value),
    }


@app.get("/di/span")
def di_span_endpoint():
    value = int(request_arg("value", "7"))
    return {"input": value, "result": span_probe_target(value)}


@app.get("/di/decorate")
def di_decorate_endpoint():
    value = int(request_arg("value", "7"))
    return {"input": value, "result": span_decoration_target(value)}


@app.get("/cpu")
def cpu_endpoint():
    from flask import request

    duration = min(max(float(request.args.get("seconds", "1.5")), 0.1), 10.0)
    deadline = time.perf_counter() + duration
    iterations = 0
    checksum = 0
    while time.perf_counter() < deadline:
        checksum = (checksum + (iterations * 31) % 9973) % 1_000_003
        iterations += 1
    return {"seconds": duration, "iterations": iterations, "checksum": checksum}


def recursive_profile_target(depth: int) -> int:
    if depth <= 0:
        return 1
    return depth + recursive_profile_target(depth - 1)


@app.get("/deep-recursion")
def deep_recursion_endpoint():
    depth = min(max(int(request_arg("depth", "48")), 5), 200)
    return {"depth": depth, "result": recursive_profile_target(depth)}


def contended_profile_target(iterations: int) -> int:
    completed = 0
    for _ in range(iterations):
        with profile_lock:
            time.sleep(0.01)
            completed += 1
    return completed


@app.get("/thread-contention")
def thread_contention_endpoint():
    workers = min(max(int(request_arg("workers", "4")), 2), 8)
    iterations = min(max(int(request_arg("iterations", "10")), 1), 50)
    threads = []
    completed = []
    for index in range(workers):
        thread = threading.Thread(
            target=lambda: completed.append(contended_profile_target(iterations)),
            name=f"ddot-parity-profile-worker-{index}",
        )
        threads.append(thread)
        thread.start()
    for thread in threads:
        thread.join()
    return {"workers": workers, "completed": sum(completed)}


async def async_wait_profile_target(delay: float) -> float:
    await asyncio.sleep(delay)
    return delay


@app.get("/async-wait")
def async_wait_endpoint():
    delay = min(max(float(request_arg("seconds", "0.25")), 0.01), 2.0)
    return {"waited_seconds": asyncio.run(async_wait_profile_target(delay))}


@app.get("/allocate")
def allocation_endpoint():
    from flask import request

    megabytes = min(max(int(request.args.get("mb", "8")), 1), 64)
    allocation_hold.append(bytearray(megabytes * 1024 * 1024))
    if len(allocation_hold) > 4:
        del allocation_hold[0]
    return {"allocated_mb": megabytes, "retained_blocks": len(allocation_hold)}


@app.get("/exceptions")
def exceptions_endpoint():
    from flask import request

    count = min(max(int(request.args.get("count", "100")), 1), 5000)
    caught = 0
    for index in range(count):
        try:
            raise ValueError(f"intentional-profile-exception-{index % 5}")
        except ValueError:
            caught += 1
    return {"caught": caught}


@app.get("/db/query")
def database_query_endpoint():
    with database_connection() as connection:
        with connection.cursor() as cursor:
            cursor.execute(
                "SELECT status, count(*) FROM orders WHERE customer_id = %s GROUP BY status",
                (42,),
            )
            rows = cursor.fetchall()
    return {"rows": [{"status": row[0], "count": row[1]} for row in rows]}


@app.get("/db/slow")
def database_slow_endpoint():
    with database_connection() as connection:
        with connection.cursor() as cursor:
            cursor.execute("SELECT pg_sleep(1.25), count(*) FROM orders")
            row = cursor.fetchone()
    return {"count": row[1], "slept_seconds": 1.25}


@app.get("/db/error")
def database_error_endpoint():
    try:
        with database_connection() as connection:
            with connection.cursor() as cursor:
                cursor.execute("SELECT * FROM validation_table_that_does_not_exist")
    except psycopg.Error as exc:
        log.warning("intentional database validation error: %s", exc)
        return {"error": exc.__class__.__name__}, 500
    return {"error": "not raised"}, 500


def request_arg(name: str, default: str) -> str:
    from flask import request

    return request.args.get(name, default)


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
