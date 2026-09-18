#!/usr/bin/env bash
set -euo pipefail

scenario="${1:?usage: $0 <scenario> [base-url]}"
base_url="${2:-http://127.0.0.1:15010}"
flush_seconds="${FLUSH_SECONDS:-20}"

request() {
  curl --fail --silent --show-error "$base_url$1" >/dev/null
}

printf 'scenario=%s\n' "$scenario"
printf 'test_data_start_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
case "$scenario" in
  di_metric)
    for value in {10..19}; do request "/probe?value=$value"; done
    ;;
  di_condition)
    for value in {1..9}; do request "/probe?value=$value"; done
    for value in {10..19}; do request "/probe?value=$value"; done
    ;;
  di_span|di_span_tag|di_log)
    for value in {20..29}; do request "/probe?value=$value"; done
    ;;
  profile_ingestion)
    for _ in {1..35}; do request "/health"; sleep 1; done
    ;;
  profile_cpu_wall|profile_endpoint|profile_hotspots)
    for _ in {1..15}; do request "/cpu?seconds=1.5"; done
    ;;
  profile_allocation)
    for _ in {1..20}; do request "/allocate?mb=8"; done
    ;;
  profile_exception)
    for _ in {1..20}; do request "/exceptions?count=500"; done
    ;;
  db_semantics|db_query_metrics)
    for _ in {1..20}; do request "/db/query"; done
    ;;
  db_query_samples|db_explain_plans|db_apm_to_dbm|db_dbm_to_apm)
    for _ in {1..10}; do request "/db/slow"; done
    ;;
  bridge)
    for _ in {1..5}; do request "/span"; done
    ;;
  *)
    printf 'unknown scenario: %s\n' "$scenario" >&2
    exit 2
    ;;
esac
printf 'test_data_end_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf 'flush_seconds=%s\n' "$flush_seconds"
sleep "$flush_seconds"
