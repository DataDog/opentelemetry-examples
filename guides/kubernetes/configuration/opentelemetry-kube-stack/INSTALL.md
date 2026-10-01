# Installation guide: the `install` script

The `install` script sets up OpenTelemetry-based Kubernetes and application monitoring with Datadog in one go: the
[OpenTelemetry Kube Stack][chart] (OpenTelemetry Operator, `cluster` and `daemon` collectors, optional host profiler),
cert-manager, the [Datadog Operator][dd-operator] and a scoped-down Datadog Agent. See
[What this deploys](README.md#what-this-deploys) for the architecture, and
[Install with values files](README.md#install-with-values-files) to perform the same installation without the script.

## Before you begin

On the machine running the script:

- **Bash**, including the stock `/bin/bash` (3.2) on macOS.
- **`kubectl`**, with its current context pointing at the target cluster (`kubectl config current-context`), and
  permissions to create namespaces, CRDs, cluster roles, and webhooks (typically `cluster-admin`).
- **`helm`**. Helm 4 is required to use the `--force-conflicts` option (Helm's server-side apply).
- **`curl`**, to download the script and its configuration files.

From Datadog:

- A Datadog [API key][dd-api-keys] and your [Datadog site][dd-site] (`datadoghq.com`, `datadoghq.eu`,
  `us3.datadoghq.com`...).
- Optionally, a Datadog [application key][dd-app-keys], to manage the Datadog Agent from
  [Fleet Automation](README.md#fleet-automation-optional).

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
`bash <(curl ...)`). Without a `.env` file, it prompts for them and offers to save them to `.env`.

To skip the credential prompts, create the `.env` file beforehand:

```sh
cat > .env <<'EOF'
export DD_SITE="datadoghq.com"
export DD_API_KEY="<your-datadog-api-key>"
# Optional, enables Fleet Automation:
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

| Prompt | Answer |
| :----- | :----- |
| Datadog Site | Your Datadog site, e.g. `datadoghq.eu`. Required. Only asked without a `.env` file. |
| Datadog API Key | Your API key (input hidden). Required. Only asked without a `.env` file. |
| Datadog Application Key | Optional (input hidden): leave empty to skip Fleet Automation. Only asked without a `.env` file. |
| Save credentials to .env file? | `y` to save the credentials to `.env` (owner-only permissions) for later runs. |
| Kubernetes cluster type | `EKS`, `GKE`, or `AKS` to auto-detect the cluster name from the cloud provider, `Other` otherwise. |
| Kubernetes Cluster Name | Only asked for `Other`. Defaults to `unknown-k8s-cluster`. Uppercase letters are lowercased and `_` replaced with `-`. |
| Deployment Environment Name | Sets `deployment.environment.name` on all telemetry. Defaults to `production`. |
| Enable the eBPF host profiler? | `y` to deploy the host profiler on every node. Defaults to `N`: its pods need eBPF privileges. |
| Egress NetworkPolicy? | Only asked with the host profiler: `s`tandard or `c`ilium on clusters enforcing NetworkPolicy, `n`one otherwise (default). |

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

Expect a `cluster` collector pod, one `daemon` collector pod per node (plus one `host-profiler` pod per node when
enabled), and one Datadog Agent pod per node next to the Datadog Operator and Cluster Agent.

After a few minutes, the cluster shows up in Datadog's Kubernetes views, under the cluster name you entered (or the one
detected on EKS, GKE, and AKS).

## 5. Instrument your applications

The script creates an `Instrumentation` custom resource, `opentelemetry-operator-system/opentelemetry-kube-stack`,
that injects SDKs into annotated pods and points them at the `daemon` collector. Add the annotation for your
application's language to its pod template, for example for Java:

```yaml
spec:
  template:
    metadata:
      annotations:
        instrumentation.opentelemetry.io/inject-java: opentelemetry-operator-system/opentelemetry-kube-stack
        resource.opentelemetry.io/service-name: my-service
```

The other languages use `inject-nodejs`, `inject-python`, and `inject-dotnet`. If you enable Datadog's APM single-step
instrumentation, exclude these pods from it, see
[Hybrid OTel + Datadog APM instrumentation](README.md#hybrid-otel--datadog-apm-instrumentation).

## Options

Run `./install --help` for the full list.

| Option | Use |
| :----- | :-- |
| `--local` | Use the configuration files next to the script instead of downloading them, e.g. to test changes from a clone of this repository. |
| `--config-url=<url>` | Download the configuration files from another GitHub folder (`https://github.com/<owner>/<repo>/tree/<branch>/<path>`) or raw base URL. |
| `--force-conflicts` | Have Helm's server-side apply take ownership of conflicting fields (Helm 4), see [Troubleshooting](#troubleshooting). |
| `<overlay-values.yaml>` | A values file merged on top of `values.yaml`: a local file, a URL, or a path relative to the configuration folder. |

For example, to also export traces to Jaeger:

```sh
./install examples/export-to-datadog-and-jaeger/values.yaml
```

## Upgrading

If the cluster runs a setup from an earlier version of this guide (the standalone `host-profiler` or the
`ddagent-kube-stack` Helm release), the `install` script stops and asks you to remove it first with the `migrate`
script, see [Migrating from a previous setup](README.md#migrating-from-a-previous-setup):

```sh
curl -fsSL -o migrate https://raw.githubusercontent.com/DataDog/opentelemetry-examples/cyrille-leclerc/use-dd-operator/guides/kubernetes/configuration/opentelemetry-kube-stack/migrate
chmod +x migrate
./migrate
```

Otherwise, re-run the script: every step is idempotent (`helm upgrade --install`, `kubectl apply`). It asks the deployment
questions again, so give the same answers to keep the same configuration. Credentials come from `.env` when saved.

## Troubleshooting

- **`UPGRADE FAILED: conflict occurred while applying object ...: conflict with "manager"`**: the OpenTelemetry
  Operator took ownership of some `Instrumentation` fields that the new configuration changes. Re-run with
  `./install --force-conflicts`.
- **`Found the legacy '...' Helm release from a previous setup`**: run `./migrate` to remove it, then re-run
  `./install`, see [Upgrading](#upgrading).
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
[dd-operator]: https://github.com/DataDog/datadog-operator
[dd-api-keys]: https://docs.datadoghq.com/account_management/api-app-keys/#api-keys
[dd-app-keys]: https://docs.datadoghq.com/account_management/api-app-keys/#application-keys
[dd-site]: https://docs.datadoghq.com/getting_started/site/
