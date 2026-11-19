# Documentation guidelines

These rules apply to Markdown documentation in this repository, including READMEs and files under `guides/`.

- Follow the [Google developer documentation style guide](https://developers.google.com/style) and the [Google Technical Writing](https://developers.google.com/tech-writing) guidelines.
- Follow the [Federal Plain Language Guidelines](https://digital.gov/guides/plain-language).
- Don't group items in threes for rhythm.
- State claims directly. Avoid negative parallelisms such as "X, not Y", "not X but Y", "it's not just X, it's Y", and "not only X but also Y". For example, rewrite "This is a configuration change, not a code change." as "This change only touches configuration."
- When another source already documents a topic, link to it. Sources to link include the [OpenTelemetry documentation](https://opentelemetry.io/docs/), the [OpenTelemetry Helm charts documentation](https://github.com/open-telemetry/opentelemetry-helm-charts), and the [Datadog documentation](https://docs.datadoghq.com/).
- Cover each topic in one file in this repository. Link to that file from other files.
- Refer to configuration that exists in this repository by file and key path, for example "`collectors.daemon.config.exporters` in `values.yaml`". Don't copy it into code blocks or link to line numbers.
