#!/usr/bin/env bash
set -euo pipefail

language="${1:?usage: set-trace-export.sh LANGUAGE ddot|native}"
mode="${2:?usage: set-trace-export.sh LANGUAGE ddot|native}"

case "$language" in
  dotnet) namespace=otel-dotnet-validation; deployment=dotnet-validation; protocol=http/json ;;
  go) namespace=otel-go-validation; deployment=go-validation; protocol=http/protobuf ;;
  java) namespace=otel-java-validation; deployment=java-validation; protocol=http/protobuf ;;
  node) namespace=otel-node-validation; deployment=node-validation; protocol=http/json ;;
  python) namespace=otel-python-validation; deployment=python-validation; protocol=http/protobuf ;;
  *) printf 'unsupported language: %s\n' "$language" >&2; exit 2 ;;
esac

if [[ "$mode" == native ]]; then
  kubectl -n "$namespace" set env deployment/"$deployment" \
    OTEL_TRACES_EXPORTER- \
    OTEL_EXPORTER_OTLP_ENDPOINT- \
    OTEL_EXPORTER_OTLP_TRACES_ENDPOINT- \
    OTEL_EXPORTER_OTLP_PROTOCOL- \
    OTEL_EXPORTER_OTLP_TRACES_PROTOCOL-
  if [[ "$language" == python ]]; then
    kubectl -n "$namespace" set env deployment/"$deployment" \
      DD_TRACE_AGENT_PROTOCOL_VERSION=v0.4
  fi
elif [[ "$mode" == ddot ]]; then
  endpoint=http://ddot-collector:4318
  trace_endpoint="$endpoint/v1/traces"
  kubectl -n "$namespace" set env deployment/"$deployment" \
    OTEL_TRACES_EXPORTER=otlp \
    OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="$trace_endpoint" \
    OTEL_EXPORTER_OTLP_TRACES_PROTOCOL="$protocol" \
    DD_TRACE_AGENT_PROTOCOL_VERSION-
  if [[ "$language" == dotnet || "$language" == java || "$language" == node ]]; then
    kubectl -n "$namespace" set env deployment/"$deployment" \
      OTEL_EXPORTER_OTLP_ENDPOINT="$endpoint"
  fi
  if [[ "$language" == java ]]; then
    kubectl -n "$namespace" set env deployment/"$deployment" \
      OTEL_EXPORTER_OTLP_PROTOCOL="$protocol"
  fi
else
  printf 'unsupported mode: %s\n' "$mode" >&2
  exit 2
fi

kubectl -n "$namespace" rollout status deployment/"$deployment" --timeout=180s
