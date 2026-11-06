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
  mkdir -p "${stub_dir}/serviceaccount" "${stub_dir}/hostfs" "${stub_dir}/otelcol"
  echo "ci-dummy-token" > "${stub_dir}/serviceaccount/token"
  echo "default" > "${stub_dir}/serviceaccount/namespace"
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout /dev/null -out "${stub_dir}/serviceaccount/ca.crt" \
    -days 1 -subj "/CN=ci" 2>/dev/null
  chmod -R a+rX "${stub_dir}"

  # Move each config into its own file, next to the image it runs on.
  # Collectors with a custom image are skipped.
  local tree values_file collector_glob default_image
  local collector_file prefix example mode doc_count i kind name custom_image
  for tree in kube-stack ddot; do
    if [[ "${tree}" = "ddot" ]]; then
      values_file=ddot/values.yaml
      collector_glob='ddot/examples/*/rendered/collector.yaml'
      prefix="ddot--"
    else
      values_file=upstream/values.yaml
      collector_glob='examples/*/rendered-ddot/collector.yaml upstream/examples/*/rendered/collector.yaml'
      prefix=""
    fi
    default_image="$(yq '.opentelemetry-operator.manager.collectorImage.repository + ":" + .opentelemetry-operator.manager.collectorImage.tag' "${values_file}")"
    for collector_file in ${collector_glob}; do
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
        echo "${default_image}" > "${work_dir}/${prefix}${example}--${mode}--${name}.image"
      done
    done
  done
  chmod -R a+rX "${work_dir}"

  local failures=0
  local config_file name image output
  for config_file in "${work_dir}"/*.yaml; do
    name="$(basename "${config_file}")"
    image="$(cat "${config_file%.yaml}.image")"
    if output=$(docker run --rm \
      --env-file "${DUMMY_ENV_FILE}" \
      -v "${stub_dir}/serviceaccount:/var/run/secrets/kubernetes.io/serviceaccount:ro" \
      -v "${stub_dir}/hostfs:/hostfs:ro" \
      -v "${stub_dir}/otelcol:/var/lib/otelcol:ro" \
      -v "${config_file}:/etc/otel/config.yaml:ro" \
      "${image}" validate --config=/etc/otel/config.yaml 2>&1); then
      echo "PASS  ${name} (${image})"
    else
      echo "FAIL  ${name} (${image})"
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
    examples/*/rendered-ddot \
    upstream/examples/*/rendered \
    datadog-agent.yaml \
    ddot/examples/*/rendered \
    ddot/datadog-agent.yaml \
    ddot/examples-datadog-agent
}

status=0
validate_collector_configs || status=1
echo
validate_manifests || status=1
exit "${status}"
