#!/usr/bin/env bash
set -euo pipefail

scenario="${1:?usage: run-python-feature-scenario.sh SCENARIO [BASE_URL]}"
base_url="${2:-http://127.0.0.1:15000}"
iterations="${ITERATIONS:-20}"

request() {
  curl --fail --silent --show-error "$base_url$1" >/dev/null
}

printf 'scenario=%s\n' "$scenario"
printf 'test_data_start_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

case "$scenario" in
  di-log)
    for ((i = 1; i <= iterations; i++)); do request "/di/log?value=$i"; done
    ;;
  di-snapshots)
    for ((i = 1; i <= iterations; i++)); do
      request "/di/snapshot-line?value=$i"
      request "/di/snapshot-method?value=$i"
    done
    ;;
  di-metrics)
    for ((i = 1; i <= iterations; i++)); do request "/di/metrics?value=$i"; done
    ;;
  di-condition)
    for ((i = 1; i <= iterations; i++)); do request "/probe?value=$i"; done
    ;;
  di-span)
    for ((i = 1; i <= iterations; i++)); do request "/di/span?value=$i"; done
    ;;
  di-decoration)
    for ((i = 1; i <= iterations; i++)); do request "/di/decorate?value=$i"; done
    ;;
  profile-identity)
    for ((i = 1; i <= iterations; i++)); do request "/deep-recursion?depth=64"; done
    ;;
  profile-cpu-wall)
    for ((i = 1; i <= iterations; i++)); do
      request "/cpu?seconds=1.5"
      request "/async-wait?seconds=0.25"
    done
    ;;
  profile-allocation)
    for ((i = 1; i <= iterations; i++)); do request "/allocate?mb=8"; done
    ;;
  profile-exceptions)
    for ((i = 1; i <= iterations; i++)); do request "/exceptions?count=500"; done
    ;;
  profile-endpoints)
    for ((i = 1; i <= iterations; i++)); do
      request "/thread-contention?workers=4&iterations=10"
      request "/deep-recursion?depth=64"
    done
    ;;
  profile-correlation)
    for ((i = 1; i <= iterations; i++)); do request "/cpu?seconds=1.5"; done
    ;;
  dbm-health|dbm-query-metrics|dbm-joins|dbm-plan|dbm-full|dbm-service)
    for ((i = 1; i <= iterations; i++)); do
      request "/db/query"
      request "/db/slow"
    done
    ;;
  *)
    printf 'unknown scenario: %s\n' "$scenario" >&2
    exit 2
    ;;
esac

printf 'test_data_end_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
