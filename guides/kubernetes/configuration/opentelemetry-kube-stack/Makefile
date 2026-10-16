TMP_DIRECTORY = ./tmp
chart_name = opentelemetry-kube-stack
EXPORTER_MODES = otlp-http ddot

REPO_ROOT := $(shell git rev-parse --show-toplevel)
CI_SCRIPTS := $(REPO_ROOT)/.github/workflows/scripts

.PHONY: validate
validate:
	@DUMMY_ENV_FILE=$(CI_SCRIPTS)/dummy-credentials.env $(CI_SCRIPTS)/validate-opentelemetry-kube-stack.sh

.PHONY: generate-base-values
generate-base-values:
	@echo "Generating values-<mode>.yaml from values.yaml.tmpl"
	@cd $(REPO_ROOT)/generate && go run .

.PHONY: generate-examples
generate-examples:
	@./render-examples $(EXPORTER_MODES)
