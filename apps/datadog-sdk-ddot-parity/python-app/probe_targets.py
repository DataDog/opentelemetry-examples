def dynamic_probe_target(value: int) -> int:
    normalized = max(0, value)
    return normalized * 2 + 1


def log_probe_target(value: int) -> int:
    return value + 1


def snapshot_line_target(value: int) -> int:
    captured = value * 3  # DDOT_PARITY_SNAPSHOT_LINE
    return captured


def snapshot_method_target(value: int) -> int:
    local_value = value * 4
    return local_value


def metric_count_target(value: int) -> int:
    return value


def metric_gauge_target(value: int) -> int:
    return value


def metric_histogram_target(value: int) -> int:
    return value


def metric_distribution_target(value: int) -> int:
    return value


def span_probe_target(value: int) -> int:
    return value * 5


def span_decoration_target(value: int) -> int:
    return value * 6
