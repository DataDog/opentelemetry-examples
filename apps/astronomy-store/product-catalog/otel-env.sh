#!/bin/sh
# Container entrypoint wrapper: adds the image's git metadata to OTEL_RESOURCE_ATTRIBUTES as OpenTelemetry
# vcs.* resource attributes, then runs the container command.
#
# Setting OTEL_RESOURCE_ATTRIBUTES with ENV in the Dockerfile doesn't work: the OpenTelemetry Operator sets
# OTEL_RESOURCE_ATTRIBUTES in the pod spec, which replaces an image ENV of the same name. This script runs
# when the container starts, after the Operator's injection, so it extends the injected value instead.
#
# Keys already present (pod spec env, Instrumentation CR, resource.opentelemetry.io/* pod annotations) are
# left unchanged. Does nothing in the ddot flow (DD_TRACE_OTEL_ENABLED=true), where the Datadog SDKs read
# DD_GIT_* instead.
#
# The same file is copied into each service directory because each image is built with its own directory
# as the build context. Keep the copies identical.
set -e

add_attr() {
    [ -n "$2" ] || return 0
    case ",${OTEL_RESOURCE_ATTRIBUTES}," in
        *",$1="*) return 0 ;;
    esac
    OTEL_RESOURCE_ATTRIBUTES="${OTEL_RESOURCE_ATTRIBUTES:+${OTEL_RESOURCE_ATTRIBUTES},}$1=$2"
}

if [ "${DD_TRACE_OTEL_ENABLED:-}" != "true" ]; then
    OTEL_RESOURCE_ATTRIBUTES="${OTEL_RESOURCE_ATTRIBUTES:-}"
    OTEL_RESOURCE_ATTRIBUTES="${OTEL_RESOURCE_ATTRIBUTES%,}"
    add_attr vcs.repository.url.full "${VCS_REPOSITORY_URL_FULL:-}"
    add_attr vcs.ref.head.revision "${VCS_REF_HEAD_REVISION:-}"
    if [ -n "${OTEL_RESOURCE_ATTRIBUTES}" ]; then
        export OTEL_RESOURCE_ATTRIBUTES
    else
        unset OTEL_RESOURCE_ATTRIBUTES
    fi
fi

exec "$@"
