#!/usr/bin/env bash
# Validates the opentelemetry-kube-stack guide.
set -euo pipefail
shopt -s inherit_errexit

yq() {
  docker run --rm -v "${PWD}:/work" -w /work mikefarah/yq:4.53.6 "$@"
}

validate_collector_configs() {
  local work_dir stub_dir
  work_dir="$(mktemp -d)"
  stub_dir="$(mktemp -d)"
  trap 'rm -rf "${work_dir}" "${stub_dir}"' RETURN

  # Add dummy files so validation can run.
  mkdir -p "${stub_dir}/serviceaccount" "${stub_dir}/hostfs"
  echo "ci-dummy-token" > "${stub_dir}/serviceaccount/token"
  echo "default" > "${stub_dir}/serviceaccount/namespace"
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout /dev/null -out "${stub_dir}/serviceaccount/ca.crt" \
    -days 1 -subj "/CN=ci" 2>/dev/null
  chmod -R a+rX "${stub_dir}"

  # Extract Collector image
  local collector_image
  collector_image="$(yq '.opentelemetry-operator.manager.collectorImage.repository + ":" + .opentelemetry-operator.manager.collectorImage.tag' values.yaml)"
  echo "Validating collector configs against ${collector_image}"

  # Move each config into its own file.
  local collector_file example doc_count i kind name
  for collector_file in examples/*/rendered/collector.yaml; do
    example="$(basename "$(dirname "$(dirname "${collector_file}")")")"
    doc_count="$(yq 'di' "${collector_file}" | tail -1)"
    for i in $(seq 0 "${doc_count}"); do
      kind="$(yq "select(di == ${i}) | .kind" "${collector_file}")"
      [[ "${kind}" = "OpenTelemetryCollector" ]] || continue
      name="$(yq "select(di == ${i}) | .metadata.name" "${collector_file}")"
      yq "select(di == ${i}) | .spec.config" "${collector_file}" \
        > "${work_dir}/${example}--${name}.yaml"
    done
  done
  chmod -R a+rX "${work_dir}"

  local failures=0
  local config_file name output
  for config_file in "${work_dir}"/*.yaml; do
    name="$(basename "${config_file}")"
    if output=$(docker run --rm \
      --env-file "${DUMMY_ENV_FILE}" \
      -v "${stub_dir}/serviceaccount:/var/run/secrets/kubernetes.io/serviceaccount:ro" \
      -v "${stub_dir}/hostfs:/hostfs:ro" \
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
    examples/*/rendered
}

status=0
validate_collector_configs || status=1
echo
validate_manifests || status=1
exit "${status}"
