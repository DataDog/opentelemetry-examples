#!/usr/bin/env bash
set -euo pipefail

if [[ $# -eq 0 ]]; then
  echo "usage: $0 <collector.yaml> [collector.yaml ...]" >&2
  exit 1
fi

status=0

for file in "$@"; do
  # select(length > 0) works around a yq quirk: an empty "X - Y" diff still
  # emits one bogus result if piped straight into `.[]`.
  out=$(
    yq eval-all '
      [.spec.config] | .[] |
      (((.receivers // {}) | keys) + ((.connectors // {}) | keys)) as $recv |
      ((.service.pipelines // {}) | to_entries)[] |
      .key as $pname |
      (((.value.receivers // []) - $recv) | select(length > 0) | .[] as $bad |
        ("undefined receiver \"" + $bad + "\" in pipeline " + $pname))
    ' "$file"
    yq eval-all '
      [.spec.config] | .[] |
      ((.processors // {}) | keys) as $proc |
      ((.service.pipelines // {}) | to_entries)[] |
      .key as $pname |
      (((.value.processors // []) - $proc) | select(length > 0) | .[] as $bad |
        ("undefined processor \"" + $bad + "\" in pipeline " + $pname))
    ' "$file"
    yq eval-all '
      [.spec.config] | .[] |
      (((.exporters // {}) | keys) + ((.connectors // {}) | keys)) as $exp |
      ((.service.pipelines // {}) | to_entries)[] |
      .key as $pname |
      (((.value.exporters // []) - $exp) | select(length > 0) | .[] as $bad |
        ("undefined exporter \"" + $bad + "\" in pipeline " + $pname))
    ' "$file"
    yq eval-all '
      [.spec.config] | .[] |
      ((.extensions // {}) | keys) as $ext |
      ((.service.extensions // []) - $ext) | select(length > 0) | .[] as $bad |
        ("undefined extension \"" + $bad + "\"")
    ' "$file"
  )
  if [[ -n "$out" ]]; then
    echo "$out" | sed "s#^#${file}: #"
    status=1
  fi
done

[[ "$status" -eq 0 ]] && echo "All collector pipeline references resolved"
exit "$status"
