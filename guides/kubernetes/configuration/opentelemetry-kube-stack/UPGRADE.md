# Upgrade guide

This guide is for Datadog engineers upgrading a cluster set up with an earlier version of this guide. A first
installation doesn't need it: follow the [installation guide](INSTALL.md) instead.

## What changed

Earlier versions of this guide installed, in the `opentelemetry-operator-system` namespace:

- the eBPF host profiler as a standalone `host-profiler` Helm release (from the `opentelemetry-collector` chart). It's
  now the optional `host-profiler` collector of the `opentelemetry-kube-stack` release.
- the Datadog Agent as the `ddagent-kube-stack` Helm release (from the `datadog/datadog` chart). It's now managed by
  the Datadog Operator, in the `datadog` namespace.

These legacy releases must not run next to the current setup: two eBPF profilers, or two Datadog Agent DaemonSets,
must not run on the same node. The `install` script refuses to run while they exist:

```
ERROR: Found the legacy 'ddagent-kube-stack' Helm release from a previous setup in the opentelemetry-operator-system namespace: run ./migrate to remove it, then re-run this script.
```

## Remove the legacy releases

Download and run the `migrate` script, then re-run the `install` script:

```sh
curl -fsSL -o migrate https://raw.githubusercontent.com/DataDog/opentelemetry-examples/feat/otel-kube-stack-ddot-installer/guides/kubernetes/configuration/opentelemetry-kube-stack/migrate
chmod +x migrate
./migrate
./install
```

`migrate` lists the legacy Helm releases it finds and asks for confirmation before uninstalling them, waiting for
their resources to be deleted. It also deletes the standalone host profiler's `host-profiler-egress` NetworkPolicy and
CiliumNetworkPolicy, which weren't part of its Helm release. Without any legacy release, it changes nothing else and
reports that there's nothing to migrate.

Between the two scripts, profiling and the features enabled by the Datadog Agent (such as Live Debugger) stop, until
`install` deploys the host profiler collector and the Datadog Operator's Agent.

## Upgrading the current setup

Once the legacy releases are removed, upgrade by re-running the `install` script: every step is idempotent (`helm upgrade --install`,
`kubectl apply`). It asks the deployment questions again, so give the same answers to keep the same configuration.
The credentials come from the `.env` file when it exists.
