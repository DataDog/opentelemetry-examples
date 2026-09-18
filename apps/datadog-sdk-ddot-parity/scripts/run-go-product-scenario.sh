#!/usr/bin/env bash
set -euo pipefail

scenario="${1:?usage: run-go-product-scenario.sh SCENARIO [BASE_URL]}"
base_url="${2:-http://127.0.0.1:15003}"
iterations="${ITERATIONS:-20}"

printf 'scenario=%s\n' "$scenario"
printf 'test_data_start_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

case "$scenario" in
  otlp-span)
    for ((iteration = 1; iteration <= iterations; iteration++)); do
      curl --fail --silent --show-error "${base_url}/span" >/dev/null
    done
    ;;
  cpu)
    for ((iteration = 1; iteration <= iterations; iteration++)); do
      curl --fail --silent --show-error "${base_url}/cpu?seconds=1.5" >/dev/null
    done
    ;;
  heap)
    for ((iteration = 1; iteration <= iterations; iteration++)); do
      curl --fail --silent --show-error "${base_url}/allocate?mb=8" >/dev/null
    done
    ;;
  goroutine)
    for ((iteration = 1; iteration <= iterations; iteration++)); do
      curl --fail --silent --show-error "${base_url}/goroutines?count=100" >/dev/null
    done
    ;;
  db-query)
    for ((iteration = 1; iteration <= iterations; iteration++)); do
      curl --fail --silent --show-error "${base_url}/db/query" >/dev/null
    done
    ;;
  db-slow)
    for ((iteration = 1; iteration <= iterations; iteration++)); do
      curl --fail --silent --show-error "${base_url}/db/slow" >/dev/null
    done
    ;;
  db-error)
    for ((iteration = 1; iteration <= iterations; iteration++)); do
      curl --silent --output /dev/null "${base_url}/db/error"
    done
    ;;
  probe-control)
    for ((iteration = 1; iteration <= iterations; iteration++)); do
      curl --fail --silent --show-error "${base_url}/probe?value=${iteration}" >/dev/null
    done
    ;;
  *)
    printf 'unknown scenario: %s\n' "$scenario" >&2
    exit 2
    ;;
esac

printf 'test_data_end_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
