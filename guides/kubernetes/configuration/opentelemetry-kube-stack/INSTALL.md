# Installation guide: the `install` script

The `install` script sets up OpenTelemetry-based Kubernetes and application monitoring for Datadog in one go. It
installs:

- the [OpenTelemetry Kube Stack][chart]: the OpenTelemetry Operator, the `cluster` and `daemon` collectors, and the
  optional host profiler, which collect the Kubernetes and application telemetry;
- [cert-manager][cm], for the OpenTelemetry Operator's admission webhook;
- the [Datadog Operator][dd-operator] and a Datadog Agent scoped down to a single role: enabling the Datadog-specific
  features of the Datadog OpenTelemetry APM SDKs, such as Live Debugger, on top of the standard OpenTelemetry
  capabilities.

See [What this deploys](README.md#what-this-deploys) for the architecture, and
[Install with values files](README.md#install-with-values-files) to perform the same installation without the script.

## Before you begin

On the machine running the script:

- **Bash**, including the stock `/bin/bash` (3.2) on macOS.
- **`kubectl`**, with its current context pointing at the target cluster (`kubectl config current-context`), and
  permissions to create namespaces, CRDs, cluster roles, and webhooks (typically `cluster-admin`).
- **`helm`**, v3 or v4.
- **`curl`**, to download the script and its configuration files.

From Datadog:

- A Datadog [API key][dd-api-keys] and your [Datadog site][dd-site] (`datadoghq.com`, `datadoghq.eu`,
  `us3.datadoghq.com`...).
- A Datadog [application key][dd-app-keys], optional but strongly recommended: it lets you manage the Datadog Agent
  from [Fleet Automation](README.md#fleet-automation-optional).

On the cluster:

- Linux nodes with kernel >= 5.10 and Kubernetes >= 1.30, only if you enable the optional eBPF host profiler.
- On platforms other than EKS, GKE, and AKS, a cluster name that follows the
  [cluster name constraints](README.md#cluster-name-constraints) (lowercase letters, digits, `-` and `.`).

## 1. Download the script

You don't need to clone the repository: by default, the script downloads its configuration files (`values.yaml`,
`datadog-agent.yaml`...) from GitHub.

```sh
curl -fsSL -o install https://raw.githubusercontent.com/DataDog/opentelemetry-examples/cyrille-leclerc/use-dd-operator/guides/kubernetes/configuration/opentelemetry-kube-stack/install
chmod +x install
```

## 2. Provide the Datadog credentials

The script reads the Datadog credentials from a `.env` file next to it (in the current directory when run with
`bash <(curl ...)`). Without a `.env` file, it prompts for them on every run.

To skip the credential prompts, create the `.env` file beforehand:

```sh
cat > .env <<'EOF'
export DD_SITE="datadoghq.com"
export DD_API_KEY="<your-datadog-api-key>"
# Optional but strongly recommended, enables Fleet Automation:
# export DD_APP_KEY="<your-datadog-application-key>"
EOF
chmod 600 .env
```

`DD_API_KEY` is required. `DD_SITE` defaults to `datadoghq.com` when missing from `.env`. Keep `.env` out of version
control (this folder's `.gitignore` ignores it).

## 3. Run the script

```sh
./install
```

The script asks the following questions:

| Prompt                         | Answer                                                                                                                     |
|:-------------------------------|:---------------------------------------------------------------------------------------------------------------------------|
| Datadog Site                   | Your Datadog site, e.g. `datadoghq.eu`. Required. Only asked without a `.env` file.                                        |
| Datadog API Key                | Your API key (input hidden). Required. Only asked without a `.env` file.                                                   |
| Datadog Application Key        | Optional but strongly recommended (input hidden): enables Fleet Automation. Only asked without a `.env` file.              |
| Kubernetes cluster type        | `EKS`, `GKE`, or `AKS` to auto-detect the cluster name from the cloud provider, `Other` otherwise.                         |
| Kubernetes Cluster Name        | Only asked for `Other`. Defaults to `unknown-k8s-cluster`. Uppercase letters are lowercased and `_` replaced with `-`.     |
| Deployment Environment Name    | Sets `deployment.environment.name` on all telemetry. Defaults to `production`.                                             |
| Enable the eBPF host profiler? | `y` to deploy the host profiler on every node. Defaults to `N`: its pods need eBPF privileges.                             |
| Egress NetworkPolicy?          | Only asked with the host profiler: `s`tandard or `c`ilium on clusters enforcing NetworkPolicy, `n`one otherwise (default). |

It then installs or upgrades every component, as described in the [Quickstart](README.md#quickstart). Some `WARNING`
lines are expected and harmless, for example when a namespace or cert-manager already exists.

## 4. Verify the installation

Check that the collectors, the OpenTelemetry Operator, and the Datadog Operator and Agent are running:

```sh
kubectl get pods --namespace opentelemetry-operator-system
kubectl get pods --namespace datadog
kubectl get opentelemetrycollectors --namespace opentelemetry-operator-system
kubectl get datadogagent --namespace datadog
```

Expect a `cluster` OpenTelemetry Collector pod, one `daemon` OpenTelemetry Collector pod per node (plus one 
`host-profiler` pod per node when enabled), and one Datadog Agent pod per node next to the Datadog Operator and Cluster
Agent.

After a few minutes, the cluster shows up in Datadog's Kubernetes views, under the cluster name you entered (or the one
detected on EKS, GKE, and AKS).

## 5. Instrument your applications

The script creates an `Instrumentation` Kubernetes custom resource, 
`opentelemetry-operator-system/opentelemetry-kube-stack`, that injects OpenTelemetry APM SDKs into annotated pods and 
points them at the `daemon` collector. Add the annotation for your application's language to its pod template, along
with the `admission.datadoghq.com/enabled` label, for example for Java:

```yaml
spec:
  template:
    metadata:
      annotations:
        instrumentation.opentelemetry.io/inject-java: opentelemetry-operator-system/opentelemetry-kube-stack
        resource.opentelemetry.io/service-name: my-service
      labels:
        # Prevent dual instrumentation by the Datadog Operator and the OpenTelemetry Operator.
        admission.datadoghq.com/enabled: "false" # Label values are strings: keep the quotes.
```

The other languages use `inject-nodejs`, `inject-python`, and `inject-dotnet`.

The `admission.datadoghq.com/enabled: "false"` label makes Datadog's admission controller skip the pod,
so that the pod is never instrumented by both the Datadog Operator and the OpenTelemetry Operator. The Datadog
Operator's APM instrumentation is disabled by default (`features.apm.instrumentation.enabled: false` in
`datadog-agent.yaml`): the label protects your pods if this configuration is later changed, for example to
auto-instrument other workloads with Datadog's single-step APM instrumentation. See
[Hybrid OTel + Datadog APM instrumentation](README.md#hybrid-otel--datadog-apm-instrumentation).

## Options

Run `./install --help` for the full list.

| Option                  | Use                                                                                                                                     |
|:------------------------|:----------------------------------------------------------------------------------------------------------------------------------------|
| `--local`               | Use the configuration files next to the script instead of downloading them, e.g. to test changes from a clone of this repository.       |
| `--config-url=<url>`    | Download the configuration files from another GitHub folder (`https://github.com/<owner>/<repo>/tree/<branch>/<path>`) or raw base URL. |
| `<overlay-values.yaml>` | A values file merged on top of `values.yaml`: a local file, a URL, or a path relative to the configuration folder.                      |

For advanced users only, for example to customize the configuration with your own values file:

```sh
./install path/to/my/overlay-values.yaml
```

## Troubleshooting

- **`Could not download ...`**: the configuration files couldn't be fetched from GitHub. Check the network access and
  the `--config-url`, or run from a clone of this repository with `--local`.
- **`No cluster name set (EKS/GKE/AKS): the Datadog Operator's Remote Configuration requires one`**: Fleet Automation
  needs an explicit cluster name. Re-run, choose `Other`, and enter the cluster name.
- **`Using cluster name '...' instead of '...'`**: the cluster name you entered was normalized to satisfy the Datadog
  Agent's constraints. Use the normalized name when looking for the cluster in Datadog.

## Uninstalling

The companion `uninstall` script removes everything the `install` script deployed. It's meant for development
clusters: it also uninstalls cert-manager and deletes the cert-manager, OpenTelemetry, and Datadog Operator CRDs,
which deletes all their custom resources cluster-wide, including ones the `install` script didn't create. It lists
what it deletes and asks for confirmation:

```sh
curl -fsSL -o uninstall https://raw.githubusercontent.com/DataDog/opentelemetry-examples/cyrille-leclerc/use-dd-operator/guides/kubernetes/configuration/opentelemetry-kube-stack/uninstall
chmod +x uninstall
./uninstall
```

[chart]: https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack
[cm]: https://cert-manager.io/docs/installation/
[dd-operator]: https://github.com/DataDog/datadog-operator
[dd-api-keys]: https://docs.datadoghq.com/account_management/api-app-keys/#api-keys
[dd-app-keys]: https://docs.datadoghq.com/account_management/api-app-keys/#application-keys
[dd-site]: https://docs.datadoghq.com/getting_started/site/
