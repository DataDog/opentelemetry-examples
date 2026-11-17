#!/usr/bin/env bash
# Validates the opentelemetry-kube-stack guide.
set -euo pipefail
shopt -s inherit_errexit

yq() {
  docker run --rm -v "${PWD}:/work" -w /work mikefarah/yq:4.53.6 "$@"
}

compare_ddot_guide_env_fragment() {
  local marker="$1"
  local selector="$2"
  local work_dir="$3"
  local guide="ddot/ADAPT-EXISTING-SETUP.md"
  local values="ddot/values.yaml"

  awk -v begin="<!-- BEGIN ${marker} -->" -v end="<!-- END ${marker} -->" '
    index($0, begin) { inside = 1; next }
    index($0, end) { exit }
    inside && $0 !~ /```/ { sub(/^  /, ""); print }
  ' "${guide}" > "${work_dir}/${marker}.yaml" || return 1
  yq -o=json '.' "${work_dir}/${marker}.yaml" > "${work_dir}/${marker}.actual.json" || return 1
  yq -o=json "[.instrumentation.env[] | select(${selector})]" \
    "${values}" > "${work_dir}/${marker}.expected.json" || return 1
  diff -u "${work_dir}/${marker}.expected.json" "${work_dir}/${marker}.actual.json"
}

validate_ddot_guide_env_fragments() {
  local work_dir status
  # The yq helper mounts the current directory into its container.
  work_dir="$(mktemp -d ./.tmp-guide-validation.XXXXXX)" || return 1
  chmod a+rx "${work_dir}" || return 1
  status=0

  compare_ddot_guide_env_fragment DDOT_MODE_ENV \
    '.name == "OTEL_TRACES_EXPORTER" or .name == "DD_TRACE_OTEL_ENABLED" or .name == "DD_LOGS_OTEL_ENABLED" or .name == "DD_METRICS_OTEL_ENABLED" or .name == "DD_TRACE_OTEL_SEMANTICS_ENABLED"' \
    "${work_dir}" || status=1
  compare_ddot_guide_env_fragment DDOT_DELTA_ENV \
    '.name == "OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE"' "${work_dir}" || status=1
  compare_ddot_guide_env_fragment DDOT_SEMCONV_ENV \
    '.name == "OTEL_SEMCONV_STABILITY_OPT_IN"' "${work_dir}" || status=1
  rm -r "${work_dir}"

  if [[ "${status}" -eq 0 ]]; then
    echo "DDOT guide environment fragments match ddot/values.yaml"
  else
    echo "::error::DDOT guide environment fragments differ from ddot/values.yaml"
  fi
  return "${status}"
}

validate_collector_configs() {
  local work_dir stub_dir
  work_dir="$(mktemp -d)"
  stub_dir="$(mktemp -d)"
  trap 'rm -rf "${work_dir}" "${stub_dir}"' RETURN

  # Add dummy files so validation can run.
  mkdir -p "${stub_dir}/serviceaccount" "${stub_dir}/hostfs" "${stub_dir}/otelcol"
  echo "ci-dummy-token" > "${stub_dir}/serviceaccount/token"
  echo "default" > "${stub_dir}/serviceaccount/namespace"
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout /dev/null -out "${stub_dir}/serviceaccount/ca.crt" \
    -days 1 -subj "/CN=ci" 2>/dev/null
  chmod -R a+rX "${stub_dir}"

  # Extract Collector image
  local collector_image
  collector_image="$(yq '.opentelemetry-operator.manager.collectorImage.repository + ":" + .opentelemetry-operator.manager.collectorImage.tag' upstream/values.yaml)"
  echo "Validating collector configs against ${collector_image}"

  # Move each config into its own file. Collectors with a custom image are skipped.
  local collector_file prefix example mode doc_count i kind name custom_image
  for collector_file in upstream/examples/*/rendered/collector.yaml ddot/examples/*/rendered/collector.yaml; do
    prefix=""
    [[ "${collector_file}" = ddot/* ]] && prefix="ddot--"
    example="$(basename "$(dirname "$(dirname "${collector_file}")")")"
    mode="$(basename "$(dirname "${collector_file}")")"
    doc_count="$(yq 'di' "${collector_file}" | tail -1)"
    for i in $(seq 0 "${doc_count}"); do
      kind="$(yq "select(di == ${i}) | .kind" "${collector_file}")"
      [[ "${kind}" = "OpenTelemetryCollector" ]] || continue
      name="$(yq "select(di == ${i}) | .metadata.name" "${collector_file}")"
      custom_image="$(yq "select(di == ${i}) | .spec.image // \"\"" "${collector_file}")"
      if [[ -n "${custom_image}" ]]; then
        echo "SKIP  ${prefix}${example}--${mode}--${name}.yaml (custom image ${custom_image})"
        continue
      fi
      yq "select(di == ${i}) | .spec.config" "${collector_file}" \
        > "${work_dir}/${prefix}${example}--${mode}--${name}.yaml"
    done
  done
  chmod -R a+rX "${work_dir}"

  local failures=0
  local config_file name output
  for config_file in "${work_dir}"/*.yaml; do
    name="$(basename "${config_file}")"
    if output=$(docker run --rm --network none \
      --env-file "${DUMMY_ENV_FILE}" \
      -v "${stub_dir}/serviceaccount:/var/run/secrets/kubernetes.io/serviceaccount:ro" \
      -v "${stub_dir}/hostfs:/hostfs:ro" \
      -v "${stub_dir}/otelcol:/var/lib/otelcol:ro" \
      -v "${config_file}:/etc/otel/config.yaml:ro" \
      "${collector_image}" validate --config=/etc/otel/config.yaml 2>&1); then
      echo "PASS  ${name}"
    else
      echo "FAIL  ${name}"
      echo "      ${output//$'\n'/$'\n'      }"
      failures=$((failures + 1))
    fi
  done

  if [[ "${failures}" -eq 0 ]]; then
    echo "All collector configs are valid"
  else
    echo "::error::${failures} collector config(s) failed validation"
    return 1
  fi
}

validate_manifests() {
  docker run --rm -v "${PWD}:/work:ro" -w /work ghcr.io/yannh/kubeconform:v0.8.0 \
    -strict -summary \
    -schema-location default \
    -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
    upstream/examples/*/rendered \
    ddot/examples/*/rendered \
    ddot/datadog-agent.yaml \
    ddot/examples-datadog-agent
}

status=0
validate_ddot_guide_env_fragments || status=1
echo
validate_collector_configs || status=1
echo
validate_manifests || status=1
exit "${status}"
