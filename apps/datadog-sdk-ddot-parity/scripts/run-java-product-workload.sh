#!/usr/bin/env bash
set -euo pipefail

scenario="${1:?usage: $0 SCENARIO [BASE_URL]}"
base_url="${2:-http://127.0.0.1:15001}"
iterations="${ITERATIONS:-20}"

request_many() {
  local path="$1"
  local allow_failure="${2:-false}"
  for ((iteration = 1; iteration <= iterations; iteration++)); do
    if [[ "$allow_failure" == "true" ]]; then
      curl --silent --output /dev/null "${base_url}${path}"
    else
      curl --fail --silent --show-error "${base_url}${path}" >/dev/null
    fi
  done
}

printf 'scenario=%s\n' "$scenario"
printf 'test_data_start_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
case "$scenario" in
  canary|profile-ingestion|profile-endpoint) request_many "/span" ;;
  di-metric) request_many "/probe/metric?value=7" ;;
  di-condition)
    request_many "/probe/condition?value=5"
    request_many "/probe/condition?value=15"
    ;;
  di-span) request_many "/probe/span?value=7" ;;
  di-span-tag) request_many "/probe/span-tag?value=7" ;;
  di-log) request_many "/probe/log?value=7" ;;
  profile-cpu) request_many "/cpu?milliseconds=1500" ;;
  profile-allocation) request_many "/allocate?mb=8" ;;
  profile-exceptions) request_many "/exceptions?count=500" ;;
  profile-hotspots) request_many "/span" ;;
  db-spans|db-query-metrics) request_many "/db/query" ;;
  db-query-samples|db-explain-plans|db-apm-to-dbm|db-dbm-to-apm)
    request_many "/db/slow"
    request_many "/db/error" true
    ;;
  *) printf 'unknown scenario: %s\n' "$scenario" >&2; exit 2 ;;
esac
printf 'test_data_end_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
