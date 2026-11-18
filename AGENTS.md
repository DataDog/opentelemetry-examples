# Documentation guidelines

These rules apply to Markdown documentation in this repository, including READMEs and files under `guides/`.

- Follow the [Google developer documentation style guide](https://developers.google.com/style) and the [Google Technical Writing](https://developers.google.com/tech-writing) guidelines.
- Follow the [Federal Plain Language Guidelines](https://digital.gov/guides/plain-language).
- Don't group items in threes for rhythm.
- State claims directly. Avoid negative parallelisms such as "X, not Y", "not X but Y", "it's not just X, it's Y", and "not only X but also Y". For example, rewrite "This is a configuration change, not a code change." as "This change only touches configuration."
- When another source already documents a topic, link to it. Sources to link include the [OpenTelemetry documentation](https://opentelemetry.io/docs/), the [OpenTelemetry Helm charts documentation](https://github.com/open-telemetry/opentelemetry-helm-charts), and the [Datadog documentation](https://docs.datadoghq.com/).
- Cover each topic in one file in this repository. Link to that file from other files. [Onboarding guides](#onboarding-guides) may restate what a reader needs to apply a rule.
- Refer to configuration that exists in this repository by file and key path, for example "`collectors.daemon.config.exporters` in `values.yaml`". Never link to line numbers. When readers need to apply or recognize the configuration, also show a short fragment in a code block: only the keys that matter, with its source named, for example "Source: `instrumentation.env` in `values.yaml`". Don't copy whole files or large sections. [Onboarding guides](#onboarding-guides) can include longer snippets.

## Terminology

- **DDOT SDK:** write "DDOT SDK" for a Datadog SDK running in OpenTelemetry mode, for example "the DDOT SDK for Java". Never write "Datadog OpenTelemetry SDK", which readers can confuse with the upstream OpenTelemetry SDKs or with the Datadog SDKs in their default mode. At the first mention in a file, describe the DDOT SDKs as "Datadog's version of the OpenTelemetry APM SDKs".
- **Version and distribution:** describe the DDOT SDKs as Datadog's "version" of the OpenTelemetry APM SDKs, never as a "distribution". Keep "distribution" for the Datadog host profiler, Datadog's distribution of the OpenTelemetry eBPF profiler.
- **OpenTelemetry SDK and OpenTelemetry Collector:** always say which version you mean, because the guidance sometimes differs between them:
  - the upstream version, from the OpenTelemetry project: "upstream OpenTelemetry SDK", for example "the upstream OpenTelemetry SDK for Go", or "upstream OpenTelemetry Collector";
  - the DDOT version: "DDOT SDK" or "DDOT Collector";
  - either of them: say so, for example "any OpenTelemetry SDK, upstream or DDOT".

  When a text uses "OpenTelemetry SDK" or "OpenTelemetry Collector" without a qualifier, read it as the upstream version.
- **Datadog SDK:** "Datadog SDK" means the `dd-trace-*` libraries in their default mode. Write "Datadog SDK", never "Datadog APM SDK".

## Onboarding guides

The guides under `guides/kubernetes/configuration/opentelemetry-kube-stack/` onboard users to DDOT, and coding agents apply some of them directly, such as `ddot/ADAPT-EXISTING-CONFIG.md`. In these guides, the success of the user or agent applying the configuration comes first, even when it means duplicating configuration or other tradeoffs. Follow these rules in them:

- **Include configuration snippets.** When a reader has to add configuration to their own setup, show it in a code block, even when the same configuration exists in `values.yaml` or `datadog-agent.yaml`. An agent may only read the guide, and a key path doesn't help a reader whose configuration has a different shape, such as raw Collector YAML or another Helm chart.
- **Make snippets complete for the rule.** A snippet can be longer than elsewhere in the repository when the rule needs it, for example the full `env` list of a container. Show it as a fragment to merge into existing configuration, and still don't copy whole files.
- **Mark the source of each snippet** that comes from a file of this repository, with a `snippet-source` comment on the line right before its code fence, and state that the files win when a snippet and its source disagree. Use one comment per source node:

  ```markdown
  <!-- snippet-source: file=guides/kubernetes/configuration/opentelemetry-kube-stack/ddot/values.yaml path=.instrumentation.env mode=subset -->
  ```

  - `file` is relative to the repository root, and `path` selects the quoted node in a yq-like syntax, such as `.defaultCRConfig.config.processors["resource_detection/env"]`. Add `snippet-path` when the snippet wraps the node in other keys.
  - `mode` is `exact` when the snippet equals the node, `subset` when it shows only some of its keys or list items, and `adapted` when it rewrites the node, for example the raw Collector configuration behind a chart preset. Placeholders such as `<CLUSTER_NAME>` match any value.
  - For `adapted` snippets, the comment stores a hash of the source node. When the check reports that the source changed, review the snippet, then store the new hash with `make check-doc-snippets ARGS=--update-hashes`.
- **Show the raw configuration behind chart presets.** When a rule relies on a kube-stack chart preset, such as `presets.kubernetesAttributes`, also show the Collector configuration it produces, for readers who don't use the chart.
- **Update snippets with their source.** Each quoted node of a source file has a comment such as `# Quoted in ddot/ADAPT-EXISTING-CONFIG.md (S-01)`, which names the guide and the rules that quote it. When you change such a node, update those snippets in the same change. Run `make check-doc-snippets` in `guides/kubernetes/configuration/opentelemetry-kube-stack/` to find the snippets that no longer match: CI runs the same check.
- **Write self-contained rules.** An agent may jump straight to one rule. Repeat in each rule the facts it needs, and link to other rules by their ID, for example "([O-04](#o-04-point-the-sdks-at-the-daemon-collector))".
- **Keep rule IDs and headings stable.** Other files and agent prompts link to them. Add new rules with the next free ID, and don't renumber or rename existing ones.
- **Spell out exact names and values.** Write environment variables, resource attributes, and configuration keys in full, in code format. When a setting accepts a fixed set of values, list the other supported values next to the example value, for example "`DD_DBM_PROPAGATION_MODE=service` (other supported values: `full`, `disabled`)".
- **State version requirements as minimums,** for example "`v2.12.0` or later". Agents take the pinned image versions from `values.yaml` and `datadog-agent.yaml`, which change more often than the guides.
- **Use placeholders such as `<CLUSTER_NAME>`** for values the reader must provide. Never show real keys or secrets.
- **Link to files of this repository with absolute URLs,** such as `https://raw.githubusercontent.com/DataDog/opentelemetry-examples/main/...`, so that an agent that only fetched the guide can open them.
- **Give a way to verify each rule,** such as a `kubectl` command or what to look for in Datadog.
