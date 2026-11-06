# Installation guide

The `install` script is the recommended way to set up OpenTelemetry-based Kubernetes and application monitoring for
Datadog. It installs the components described in [What this deploys](README.md#what-this-deploys), and
[cert-manager][cm] if it isn't already installed. To install manually, see
[Install without the `install` script](#install-without-the-install-script).

If you already run an OpenTelemetry Collector, the OpenTelemetry Operator, OpenTelemetry SDKs, or a Datadog Agent, see
[Adapt an existing configuration for DDOT](ADAPT-EXISTING-CONFIG.md).

Contents:

- [Before you begin](#before-you-begin)
- [1. Download the script](#1-download-the-script)
- [2. Configure the installation](#2-configure-the-installation)
- [3. Run the script](#3-run-the-script)
- [4. Verify the installation](#4-verify-the-installation)
  - [Verify on the Kubernetes cluster](#verify-on-the-kubernetes-cluster)
  - [Verify in Datadog](#verify-in-datadog)
- [5. Instrument your applications](#5-instrument-your-applications)
  - [Go applications](#go-applications)
- [Options of the `install` script](#options-of-the-install-script)
- [Customize the installation](#customize-the-installation)
  - [Use an overlay values file for the OpenTelemetry Kube Stack Helm chart](#use-an-overlay-values-file-for-the-opentelemetry-kube-stack-helm-chart)
  - [Copy the configuration files locally and use `--local`](#copy-the-configuration-files-locally-and-use---local)
- [Changes to the Kubernetes cluster](#changes-to-the-kubernetes-cluster)
- [Troubleshooting](#troubleshooting)
- [Uninstalling](#uninstalling)
- [Install without the `install` script](#install-without-the-install-script)

## Before you begin

On the machine running the script:

- **Bash** 3.2 or later, on Linux or macOS (the stock `/bin/bash` 3.2 on macOS works).
- **`kubectl`**, with its current context pointing at the target Kubernetes cluster (`kubectl config current-context`),
  and permissions to create namespaces, CRDs, cluster roles, and webhooks (typically `cluster-admin`).
- **`helm`**, v3 or v4.
- **`curl`**, to download the script and its configuration files.

From Datadog:

- A Datadog [API key][dd-api-keys] and your [Datadog site][dd-site], for example `datadoghq.com` or `datadoghq.eu`.

On the Kubernetes cluster:

- If you enable the optional eBPF host profiler: Linux nodes with kernel >= 5.10, and Kubernetes >= 1.30.
- On platforms other than EKS, GKE, and AKS, a Kubernetes cluster name that follows the
  [Kubernetes cluster name constraints](README.md#kubernetes-cluster-name-constraints).

## 1. Download the script

You don't need to clone the repository: by default, the script downloads its configuration files (`values.yaml`,
`datadog-agent.yaml`...) from GitHub.

```sh
curl -fsSL -o install https://raw.githubusercontent.com/DataDog/opentelemetry-examples/bddea96e59ce0cb8006068df17aa50839f1d439d/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/install
chmod +x install
```

## 2. Configure the installation

The script reads its configuration from environment variables and never prompts, so it can run unattended, for example
in CI:

| Variable                          | Description                                                                                                                         | Default                                                     |
|:----------------------------------|:------------------------------------------------------------------------------------------------------------------------------------|:------------------------------------------------------------|
| `DD_API_KEY`                      | Datadog API key.                                                                                                                    | None (required)                                             |
| `DD_SITE`                         | Datadog site, for example `datadoghq.eu`.                                                                                           | `datadoghq.com`                                             |
| `K8S_CLUSTER_TYPE`                | Kubernetes platform: `eks` (EKS), `gcp` (GKE), `aks` (AKS), or `other`. `eks`, `gcp`, and `aks` auto-detect the cluster name from the cloud provider. | `other`                                                     |
| `K8S_CLUSTER_NAME`                | Kubernetes cluster name. For `eks`, `gcp`, and `aks`, overrides the auto-detected name. Must follow the [Kubernetes cluster name constraints](README.md#kubernetes-cluster-name-constraints). | Auto-detected for `eks`, `gcp`, and `aks`; `unknown-k8s-cluster` for `other` |
| `DEPLOYMENT_ENVIRONMENT_NAME`     | Value of the `deployment.environment.name` resource attribute, set on all telemetry.                                               | `production`                                                |
| `DD_HOST_PROFILER_ENABLED`        | Deploy the eBPF host profiler on every node: `true` or `false`. Its pods need eBPF privileges. See [Host profiler (optional)](README.md#host-profiler-optional). | `false`                                                     |
| `DD_HOST_PROFILER_NETWORK_POLICY` | Host profiler egress NetworkPolicy: `standard` or `cilium` on Kubernetes clusters enforcing NetworkPolicy, `none` otherwise. | `none`                                                      |

The script validates the variables before making any change and prints the resulting configuration. For example:

```sh
export DD_API_KEY="<your-datadog-api-key>"
export DD_SITE="datadoghq.com"
export K8S_CLUSTER_TYPE=eks
```

Alternatively, keep the configuration in a `.env` file, starting from [`.env.example`](.env.example):

```sh
curl -fsSL -o .env https://raw.githubusercontent.com/DataDog/opentelemetry-examples/bddea96e59ce0cb8006068df17aa50839f1d439d/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/.env.example
chmod 600 .env
# Edit .env
source .env
```

Keep `.env` out of version control (the repository's `.gitignore` ignores it).

## 3. Run the script

```sh
./install
```

The script:

- creates the `opentelemetry-operator-system` and `datadog` namespaces;
- creates the `datadog-secret` secret in both namespaces;
- installs cert-manager, unless it's already installed;
- installs or upgrades the Datadog Operator in the `datadog` namespace;
- deploys the Datadog Agent through a `DatadogAgent` custom resource in the `datadog` namespace;
- installs or upgrades the OpenTelemetry Kube Stack Helm chart, optionally with the `host-profiler` collector.

Some `WARNING` lines are expected and harmless, for example when a namespace already exists.

See [Changes to the Kubernetes cluster](#changes-to-the-kubernetes-cluster) for the exact `kubectl` and `helm` commands.

## 4. Verify the installation

### Verify on the Kubernetes cluster

Check that the collectors, the OpenTelemetry Operator, and the Datadog Operator and Agent are running:

```sh
kubectl get pods --namespace opentelemetry-operator-system
kubectl get pods --namespace datadog
kubectl get opentelemetrycollectors --namespace opentelemetry-operator-system
kubectl get datadogagent --namespace datadog
```

Expect one `cluster` collector pod, one `daemon` collector pod per node (plus one `host-profiler` pod per node when
enabled), and one Datadog Agent pod per node, next to the Datadog Operator and Cluster Agent.

### Verify in Datadog

After a few minutes, the installation shows up in Datadog under the Kubernetes cluster name you set, or the one detected
on EKS, GKE, and AKS:

- **[Fleet Automation][dd-fleet-automation]** lists the Datadog Agents and the `cluster` and `daemon` OpenTelemetry
  Collectors, which report to it through their [Datadog extension][dd-otel-extension]. Check that it lists one Datadog
  Agent and one `daemon` collector per node, and one `cluster` collector.
- **The default OpenTelemetry Collector dashboard** (in **Dashboards**, search for `OpenTelemetry Collector`) shows
  the collectors' [health metrics][dd-otel-collector-health]. Check that they receive and export telemetry without
  refused or failed data.
- The Kubernetes cluster shows up in Datadog's Kubernetes views.

## 5. Instrument your applications

Add the annotation for your application's language and the `admission.datadoghq.com/enabled` label to its pod
template. The annotation references the `opentelemetry-operator-system/opentelemetry-kube-stack` `Instrumentation`
custom resource, which instruments the pod with the DDOT SDKs and points them at the `daemon` collector. For example,
for Java:

```yaml
spec:
  template:
    metadata:
      annotations:
        instrumentation.opentelemetry.io/inject-java: opentelemetry-operator-system/opentelemetry-kube-stack
        resource.opentelemetry.io/service.name: my-service
      labels:
        # Prevent dual instrumentation by the Datadog Operator and the OpenTelemetry Operator.
        admission.datadoghq.com/enabled: "false" # Label values are strings: keep the quotes.
```

The other languages use `inject-nodejs`, `inject-python`, and `inject-dotnet`.

For applications that bundle the DDOT SDK in their container image, use the `inject-sdk` annotation, as for
[Go applications](#go-applications).

For what the `admission.datadoghq.com/enabled` label does, see
[Hybrid OTel + Datadog APM instrumentation](README.md#hybrid-otel--datadog-apm-instrumentation).

### Go applications

Go apps can't be auto-instrumented at runtime by injection. Instrument them at build time instead, with
**compile-time auto-instrumentation**: it auto-instruments popular libraries (`net/http`, gRPC, database drivers...),
and your code uses only the OpenTelemetry APIs to add custom spans, attributes, or metrics. Use the tool that matches
your SDK:

- **Upstream OpenTelemetry SDK for Go:** [OpenTelemetry Go compile-time instrumentation][otel-go-compile-time].
- **DDOT SDK for Go**, the Datadog OpenTelemetry SDK for Go: [Orchestrion][orchestrion].

Then add the `inject-sdk` annotation to the pod template. For this annotation, the OpenTelemetry Operator injects only
the OpenTelemetry configuration, as `OTEL_*` environment variables such as the `daemon` collector's endpoint and the
resource attributes:

```yaml
spec:
  template:
    metadata:
      annotations:
        instrumentation.opentelemetry.io/inject-sdk: opentelemetry-operator-system/opentelemetry-kube-stack
        resource.opentelemetry.io/service.name: my-service
      labels:
        # Prevent dual instrumentation by the Datadog Operator and the OpenTelemetry Operator.
        admission.datadoghq.com/enabled: "false" # Label values are strings: keep the quotes.
```

## Options of the `install` script

Run `./install --help` for the full list.

| Option                  | Use                                                                                                                                     |
|:------------------------|:----------------------------------------------------------------------------------------------------------------------------------------|
| `--local`               | Use the configuration files next to the script instead of downloading them, for example to test changes from a clone of this repository. |
| `--config-url=<url>`    | Download the configuration files from another GitHub folder (`https://github.com/<owner>/<repo>/tree/<branch>/<path>`) or raw base URL. |
| `<overlay-values.yaml>` | A values file merged on top of the reference `values.yaml`: a local file, a URL, or a path relative to the configuration folder.       |

## Customize the installation

Customize the installation in one of two ways:

- [**An overlay values file**](#use-an-overlay-values-file-for-the-opentelemetry-kube-stack-helm-chart) for the
  OpenTelemetry Kube Stack Helm chart, for targeted changes to the collectors such as exporting to another
  observability backend. You keep receiving the updates to the reference `values.yaml`.
- [**Local copies of the configuration files**](#copy-the-configuration-files-locally-and-use---local), used with the
  `--local` option. This gives full control, including over the `DatadogAgent` custom resource (`datadog-agent.yaml`),
  but you maintain your copies yourself.

### Use an overlay values file for the OpenTelemetry Kube Stack Helm chart

Values in the overlay values file take precedence over the reference `values.yaml`. The overlay values file can be a
local file, a URL, or a path relative to the configuration folder, which by default is this guide's folder on GitHub.

For example, to export the traces of the `daemon` collector to a [Jaeger][jaeger] instance in addition to Datadog,
use [`examples/export-to-datadog-and-jaeger/values.yaml`](examples/export-to-datadog-and-jaeger/values.yaml):

```yaml
collectors:
  daemon:
    config:
      exporters:
        otlp/jaeger:
          endpoint: jaeger.jaeger.svc.cluster.local:4317
          tls:
            insecure: true

      service:
        pipelines:
          traces:
            exporters: [datadog/connector, otlp/jaeger]
```

Pass it to the script as a path relative to the configuration folder, or as your own local copy:

```sh
./install examples/export-to-datadog-and-jaeger/values.yaml
# or
./install my-overlay-values.yaml
```

- Helm merges maps but **replaces lists**: the overlay's `exporters` list replaces the reference one, so it must repeat
  `datadog/connector` to keep traces flowing to Datadog. Check the current lists in `values.yaml`
  (`collectors.daemon.config.service.pipelines`) before overriding one.
- Adapt the endpoint and TLS settings to your Jaeger instance. This example expects the OTLP gRPC endpoint of a Jaeger
  instance in the `jaeger` namespace, without TLS.
- The same pattern applies to any OTLP-compatible observability backend, and to the `logs` and `metrics` pipelines.

### Copy the configuration files locally and use `--local`

Download the script and its configuration files from the same commit into one folder:

```sh
CONFIG_RAW_URL=https://raw.githubusercontent.com/DataDog/opentelemetry-examples/bddea96e59ce0cb8006068df17aa50839f1d439d/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot
for file in install values.yaml datadog-agent.yaml host-profiler-rbac-values.yaml \
    host-profiler-network-policy.yaml host-profiler-cilium-network-policy.yaml; do
  curl -fsSL -o "$file" "$CONFIG_RAW_URL/$file"
done
chmod +x install
```

Edit the local files, mainly:

- `values.yaml`, the values file of the OpenTelemetry Kube Stack Helm chart: the collectors, their pipelines and
  resources, the `Instrumentation` custom resource...
- `datadog-agent.yaml`, the `DatadogAgent` custom resource from which the Datadog Operator deploys the Datadog Agent:
  its features, resources...

Run the script with `--local`, optionally with an overlay values file relative to the current directory:

```sh
./install --local
# or
./install --local my-overlay-values.yaml
```

- Keep the `<CLUSTER_NAME>` and `<DD_SITE>` placeholders of `datadog-agent.yaml`: the script substitutes them.
- Keep the `opentelemetry-operator-system` and `datadog` namespaces: the script and the DDOT SDK instrumentation
  annotations rely on them.
- Your local copies don't receive this guide's updates. Compare them with the newer versions when you upgrade.

## Changes to the Kubernetes cluster

The `install` script runs the following commands on the Kubernetes cluster, in this order. It never deletes anything, so
re-running it is safe.

| # | Command                                                                                                                                              | When                                                                                    | Effect                                                                                                                                                                       |
|:--|:-----------------------------------------------------------------------------------------------------------------------------------------------------|:----------------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 1 | `kubectl create namespace opentelemetry-operator-system`                                                                                             | Always (warns and continues on failure)                                                 | Creates the namespace of the OpenTelemetry Kube Stack.                                                                                                                       |
| 2 | `kubectl create namespace datadog`                                                                                                                   | Always (warns and continues on failure)                                                 | Creates the namespace of the Datadog Operator and Agent.                                                                                                                     |
| 3 | `kubectl create secret generic datadog-secret ... --dry-run=client -o yaml \| kubectl apply -f -`                                                    | Always, once per namespace                                                              | Creates or updates the `datadog-secret` secret with `api-key` and `dd-site`.                                                                                                 |
| 4 | `helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set crds.enabled=true`                                | Only when the `certificates.cert-manager.io` CRD is missing (warns and continues on failure) | Installs cert-manager and its CRDs in the `cert-manager` namespace.                                                                                                     |
| 5 | `helm upgrade --install datadog-operator datadog/datadog-operator --version 2.28.1 --namespace datadog ...`                                          | Always                                                                                  | Installs or upgrades the Datadog Operator and its CRDs.                                                                                                                      |
| 6 | `kubectl apply --namespace datadog -f -` (`datadog-agent.yaml`, with the Kubernetes cluster name and site substituted)                               | Always                                                                                  | Creates or updates the `DatadogAgent` custom resource, from which the Datadog Operator deploys the Datadog Agent DaemonSet and the Cluster Agent.                           |
| 7 | `kubectl apply -f host-profiler-network-policy.yaml` or `kubectl apply -f host-profiler-cilium-network-policy.yaml`                                  | Only with `DD_HOST_PROFILER_ENABLED=true` and `DD_HOST_PROFILER_NETWORK_POLICY=standard` or `cilium` (warns and continues on failure) | Creates or updates the `opentelemetry-kube-stack-host-profiler-egress` NetworkPolicy or CiliumNetworkPolicy in the `opentelemetry-operator-system` namespace. |
| 8 | `helm upgrade --install opentelemetry-kube-stack open-telemetry/opentelemetry-kube-stack --version 0.26.1 --namespace opentelemetry-operator-system ...` | Always                                                                                  | Installs or upgrades the OpenTelemetry Operator and its CRDs, the `cluster` and `daemon` collectors, and the `Instrumentation` custom resource, plus the `host-profiler` collector and its RBAC (`host-profiler-rbac-values.yaml`) when enabled. |

Before making any change, the script also runs read-only checks and stops on a conflict:

- `helm status` for the legacy `host-profiler` and `ddagent-kube-stack` Helm releases of a previous setup. If found,
  remove them first with `helm uninstall <release> --namespace opentelemetry-operator-system --wait`.
- `kubectl get`, when the host profiler is enabled, for an `opentelemetry-kube-stack-host-profiler-egress`
  NetworkPolicy or CiliumNetworkPolicy of a kind other than the selected one, left by a previous run. If found, the
  script asks you to delete it, because both policies select the same pods.
- `kubectl get crd certificates.cert-manager.io`, to skip the cert-manager installation when it's already installed.

On the machine running the script, it adds the `open-telemetry`, `jetstack`, and `datadog` Helm repositories and runs
`helm repo update`. It downloads the configuration files to a temporary directory that it deletes on exit.

## Troubleshooting

- **`Could not download ...`**: the script couldn't fetch the configuration files from GitHub. Check the network access
  and the `--config-url`, or run from a clone of this repository with `--local`.
- **`A host-profiler egress ... from a previous installation conflicts with DD_HOST_PROFILER_NETWORK_POLICY=...`**: a previous run
  applied the other kind of host-profiler egress NetworkPolicy. Run the `kubectl delete` command in the error message,
  then re-run the script.

## Uninstalling

The `uninstall` script removes everything the `install` script deployed. Use it on development Kubernetes clusters only:
it also uninstalls cert-manager and deletes the cert-manager, OpenTelemetry, and Datadog Operator CRDs. Deleting the
CRDs deletes all their custom resources cluster-wide, including ones the `install` script didn't create. The script
lists what it deletes and asks for confirmation:

```sh
curl -fsSL -o uninstall https://raw.githubusercontent.com/DataDog/opentelemetry-examples/bddea96e59ce0cb8006068df17aa50839f1d439d/guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/uninstall
chmod +x uninstall
./uninstall
```

## Install without the `install` script

> [!IMPORTANT]
> Follow these steps only if you can't use the [`install` script](#1-download-the-script), for example in GitOps
> pipelines or other tooling.

Run these commands from a clone of this repository, in this folder. They use its `values.yaml`, `examples/`, and
`datadog-agent.yaml` files.

Set the Datadog credentials:

```sh
export DD_API_KEY="<your-datadog-api-key>"
export DD_SITE="datadoghq.com" # Use your Datadog site when different.
```

Create the namespaces and a `datadog-secret` secret in each: `opentelemetry-operator-system` for the collectors' Datadog
exporter, and `datadog` for the `DatadogAgent` custom resource, which can only reference a secret in its own namespace:

```sh
kubectl create namespace opentelemetry-operator-system \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace datadog \
  --dry-run=client -o yaml | kubectl apply -f -

for NS in opentelemetry-operator-system datadog; do
  kubectl create secret generic datadog-secret \
    --namespace "$NS" \
    --from-literal="api-key=$DD_API_KEY" \
    --from-literal="dd-site=$DD_SITE" \
    --dry-run=client -o yaml | kubectl apply -f -
done
```

Install cert-manager:

```sh
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true \
  --wait \
  --timeout 5m
```

Copy the example for your Kubernetes platform to `deployment/values.yaml`, then set its deployment environment. The
EKS, GKE, and AKS examples enable the platform's resource detector, which detects `k8s.cluster.name`:

```sh
mkdir -p deployment

# Choose one:
cp examples/eks-deployment/values.yaml deployment/values.yaml
cp examples/gcp-deployment/values.yaml deployment/values.yaml
cp examples/aks-deployment/values.yaml deployment/values.yaml
```

On other Kubernetes platforms, copy the manual Kubernetes cluster name example, then replace `my-k8s-cluster` and
`production` with the Kubernetes cluster name and deployment environment. `DD_SITE` still comes from `datadog-secret`.

```sh
mkdir -p deployment
cp examples/manually-set-k8s-cluster-name/values.yaml deployment/values.yaml
```

The copied file is an overlay for this folder's base `values.yaml`. Add any deployment-specific configuration to
`./deployment/values.yaml`. Install or upgrade the chart with both files:

```sh
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
helm upgrade --install opentelemetry-kube-stack \
  open-telemetry/opentelemetry-kube-stack \
  --version 0.26.1 \
  --namespace opentelemetry-operator-system \
  --values ./values.yaml \
  --values ./deployment/values.yaml
```

To also enable the `host-profiler` collector, run this command instead:

```sh
helm upgrade --install opentelemetry-kube-stack \
  open-telemetry/opentelemetry-kube-stack \
  --version 0.26.1 \
  --namespace opentelemetry-operator-system \
  --set collectors.host-profiler.enabled=true \
  --values ./values.yaml \
  --values ./host-profiler-rbac-values.yaml \
  --values ./deployment/values.yaml
```

On Kubernetes clusters enforcing NetworkPolicy, also apply one of:

```sh
kubectl apply -f ./host-profiler-network-policy.yaml          # any enforcing CNI
# or
kubectl apply -f ./host-profiler-cilium-network-policy.yaml   # Cilium, FQDN-scoped egress
```

Finally, install the [Datadog Operator][dd-operator] and apply the `DatadogAgent` custom resource, substituting the
`<CLUSTER_NAME>` and `<DD_SITE>` placeholders in `datadog-agent.yaml`. Set `K8S_CLUSTER_NAME` to the Kubernetes cluster
name from `deployment/values.yaml`, following the
[Kubernetes cluster name constraints](README.md#kubernetes-cluster-name-constraints). On EKS, GKE, and AKS, leave it
empty: the cluster name is auto-detected.

```sh
export K8S_CLUSTER_NAME="my-k8s-cluster" # empty string on EKS/GKE/AKS

helm repo add datadog https://helm.datadoghq.com
helm repo update
OPERATOR_HELM_ARGS=(--set apiKeyExistingSecret=datadog-secret --set-string site="$DD_SITE")
if [[ -n "$K8S_CLUSTER_NAME" ]]; then
  OPERATOR_HELM_ARGS+=(--set-string clusterName="$K8S_CLUSTER_NAME")
fi
helm upgrade --install datadog-operator \
  datadog/datadog-operator \
  --version 2.28.1 \
  --namespace datadog \
  "${OPERATOR_HELM_ARGS[@]}" \
  --wait --timeout 5m

# On EKS/GKE/AKS (empty K8S_CLUSTER_NAME), drop the clusterName line so the Agent auto-detects it.
# Likewise, only set the Agent hostname from the Kubernetes node name and the Kubernetes cluster name on non-cloud
# Kubernetes clusters.
if [[ -n "$K8S_CLUSTER_NAME" ]]; then
  CLUSTER_NAME_SED_ARGS=(-e "s|<CLUSTER_NAME>|$K8S_CLUSTER_NAME|" -e "/<HOSTNAME_FROM_NODE_NAME:/d")
else
  CLUSTER_NAME_SED_ARGS=(-e "/<CLUSTER_NAME>/d" -e "/<HOSTNAME_FROM_NODE_NAME:BEGIN>/,/<HOSTNAME_FROM_NODE_NAME:END>/d")
fi

sed \
  "${CLUSTER_NAME_SED_ARGS[@]}" \
  -e "s|<DD_SITE>|$DD_SITE|" \
  ./datadog-agent.yaml \
  | kubectl apply --namespace datadog -f -
```

[cm]: https://cert-manager.io/docs/installation/
[dd-operator]: https://github.com/DataDog/datadog-operator
[dd-api-keys]: https://docs.datadoghq.com/account_management/api-app-keys/#api-keys
[dd-site]: https://docs.datadoghq.com/getting_started/site/
[jaeger]: https://www.jaegertracing.io/
[otel-go-compile-time]: https://opentelemetry.io/docs/zero-code/go/compile-time/
[orchestrion]: https://github.com/DataDog/orchestrion
[dd-fleet-automation]: https://docs.datadoghq.com/agent/fleet_automation/
[dd-otel-collector-health]: https://docs.datadoghq.com/opentelemetry/integrations/collector_health_metrics/
[dd-otel-extension]: https://docs.datadoghq.com/opentelemetry/integrations/datadog_extension/
