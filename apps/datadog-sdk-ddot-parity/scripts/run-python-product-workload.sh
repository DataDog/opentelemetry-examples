#!/usr/bin/env bash
set -euo pipefail

base_url="${1:-http://127.0.0.1:15000}"
iterations="${ITERATIONS:-20}"

printf 'test_data_start_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
for ((iteration = 1; iteration <= iterations; iteration++)); do
  curl --fail --silent --show-error "${base_url}/probe?value=${iteration}" >/dev/null
  curl --fail --silent --show-error "${base_url}/cpu?seconds=1.5" >/dev/null
  curl --fail --silent --show-error "${base_url}/allocate?mb=8" >/dev/null
  curl --fail --silent --show-error "${base_url}/exceptions?count=250" >/dev/null
  curl --fail --silent --show-error "${base_url}/db/query" >/dev/null
  curl --fail --silent --show-error "${base_url}/db/slow" >/dev/null
  curl --silent --output /dev/null "${base_url}/db/error"
  curl --fail --silent --show-error "${base_url}/span" >/dev/null
done
printf 'test_data_end_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
