#!/usr/bin/env bash
# Exercises the installer with local configuration and stubbed cluster commands.
set -euo pipefail

root=$(cd "$(dirname "$0")/../../.." && pwd)
guide="$root/guides/kubernetes/configuration/opentelemetry-kube-stack"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

cat >"$work/kubectl" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == --context ]]; then
    shift 2
fi
case "$1 $2 ${3:-}" in
    'config current-context ') echo test ;;
    'config view --minify') echo https://test.invalid ;;
    'get configmap '*)
        case "$SCENARIO" in
            migration|upgrade|conflict|lost-cilium|disabled) cat "$CALLS.options" ;;
            legacy-upgrade) echo DD_INSTALL_FLOW=upstream ;;
        esac
        ;;
    'get crd '*) exit 1 ;;
    'api-resources --api-group=cilium.io --namespaced')
        echo "discover policies" >>"$CALLS"
        if [[ "$SCENARIO" == standard || "$SCENARIO" == existing-egress || "$SCENARIO" == lost-cilium ]]; then
            echo ciliumclusterwidenetworkpolicies.cilium.io
        else
            echo ciliumnetworkpolicies.cilium.io
            if [[ "$SCENARIO" == partial-discovery ]]; then
                exit 1
            fi
        fi
        ;;
    'get networkpolicies.v1.networking.k8s.io opentelemetry-kube-stack-host-profiler-egress') ;;
    'get ciliumnetworkpolicies.v2.cilium.io opentelemetry-kube-stack-host-profiler-egress')
        if [[ "$SCENARIO" == dry-run-denied || "$SCENARIO" == conflict ]]; then
            echo ciliumnetworkpolicy.cilium.io/opentelemetry-kube-stack-host-profiler-egress
        fi
        ;;
    'get networkpolicies.v1.networking.k8s.io --namespace')
        if [[ "$SCENARIO" == existing-egress || "$SCENARIO" == cilium ]]; then
            echo default-deny
        fi
        ;;
    'get networkpolicy '*|'get ciliumnetworkpolicy '*) ;;
    'auth can-i '*) echo yes ;;
    'create '*)
        if [[ "$2" == configmap && "$3" == opentelemetry-kube-stack-install-options ]]; then
            echo 'kind: ConfigMap'
            for arg; do
                case "$arg" in
                    --from-literal=*) printf '%s\n' "${arg#--from-literal=}" ;;
                esac
            done
        elif [[ "${2:-}" != -f ]]; then
            echo "kind: Unrelated"
        else
            cat >/dev/null
        fi
        ;;
    'label '*|'annotate '*) cat ;;
    'apply --dry-run=server -f'|'apply -f -')
        manifest=$(cat)
        case "$manifest" in
            *'kind: ConfigMap'*) printf '%s\n' "$manifest" | tail -n +2 >"$CALLS.options"; exit 0 ;;
            *'kind: CiliumNetworkPolicy'*) kind=cilium ;;
            *'kind: NetworkPolicy'*) kind=standard ;;
            *) exit 0 ;;
        esac
        if [[ "$2" == --dry-run=server ]]; then
            echo "dry-run $kind" >>"$CALLS"
            if [[ "$SCENARIO" == dry-run-denied ]]; then
                echo 'Policy rejected by admission' >&2
                exit 1
            fi
        else
            echo "apply $kind" >>"$CALLS"
        fi
        ;;
    'patch configmap opentelemetry-kube-stack-install-options')
        case "${*: -1}" in
            '{"data":{"HOST_PROFILER_REMEMBERED_KIND":"standard"}}') kind=standard ;;
            '{"data":{"HOST_PROFILER_REMEMBERED_KIND":"cilium"}}') kind=cilium ;;
            *) exit 1 ;;
        esac
        echo "record $kind" >>"$CALLS"
        options=$(sed '/^HOST_PROFILER_REMEMBERED_KIND=/d' "$CALLS.options")
        printf '%s\nHOST_PROFILER_REMEMBERED_KIND=%s\n' "$options" "$kind" >"$CALLS.options"
        ;;
    *) echo "Unexpected kubectl command: $*" >&2; exit 1 ;;
esac
STUB

printf '#!/usr/bin/env bash\nexit 0\n' >"$work/helm"
chmod +x "$work/kubectl" "$work/helm"

run_case() {
    local scenario=$1 expected=$2 message=$3 enabled=${4:-true} mode=${5:-}
    local output actual
    : >"$work/calls"
    output=$(
        unset DD_HOST_PROFILER_NETWORK_POLICY
        if [[ -n "$mode" ]]; then
            export DD_HOST_PROFILER_NETWORK_POLICY="$mode"
        fi
        PATH="$work:$PATH" SCENARIO="$scenario" CALLS="$work/calls" \
            TESTING_BASE_URL="file://$guide" DD_API_KEY=offline-test DD_INSTALL_FLOW=upstream \
            DD_AGENT_ENABLED=false DD_INSTRUMENTATION_TELEMETRY_ENABLED=false DD_HOST_PROFILER_ENABLED="$enabled" \
            "$BASH" "$guide/install" 2>&1
    ) || {
        printf 'FAIL %s:\n%s\n' "$scenario" "$output" >&2
        exit 1
    }
    actual=$(<"$work/calls")
    if [[ "$actual" != "$expected" || "$output" != *"$message"* ]]; then
        printf 'FAIL %s\nExpected calls:\n%s\nActual calls:\n%s\nOutput:\n%s\n' "$scenario" "$expected" "$actual" "$output" >&2
        exit 1
    fi
    echo "PASS $scenario"
}

cilium_calls=$'discover policies\ndry-run cilium\napply cilium\nrecord cilium'
standard_calls=$'discover policies\ndry-run standard\napply standard\nrecord standard'

# Fresh installs and discovery failures.
run_case cilium "$cilium_calls" 'Applied the cilium Host Profiler policy.'
run_case standard "$standard_calls" 'Applied the standard Host Profiler policy.'
run_case existing-egress 'discover policies' 'Skipping the standard Host Profiler policy because other Kubernetes egress policies exist in opentelemetry-operator-system: default-deny'
run_case partial-discovery 'discover policies' 'Could not discover the Cilium policy API'

# Observed Cilium must survive a failed dry-run and a disabled run that requests standard.
run_case dry-run-denied $'discover policies\ndry-run cilium' 'without trying a different kind'
run_case disabled '' 'Installing OpenTelemetry Kube Stack' false standard
run_case lost-cilium 'discover policies' 'its policy API is unavailable' true auto

# Explicit migration updates the remembered kind even if there is no live policy on the next run.
run_case migration $'apply standard\nrecord standard' 'Applied the standard Host Profiler policy.' true standard
run_case upgrade "$standard_calls" 'Applied the standard Host Profiler policy.' true auto
run_case conflict 'discover policies' 'Conflicting Host Profiler policy kinds.'

# Legacy installs without a mode keep none.
run_case legacy-upgrade '' 'Installing OpenTelemetry Kube Stack'
