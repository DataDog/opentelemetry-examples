TMP_DIRECTORY = ./tmp
chart_name = opentelemetry-kube-stack

REPO_ROOT := $(shell git rev-parse --show-toplevel)
CI_SCRIPTS := $(REPO_ROOT)/.github/workflows/scripts

.PHONY: validate
validate:
	@DUMMY_ENV_FILE=$(CI_SCRIPTS)/dummy-credentials.env $(CI_SCRIPTS)/validate-opentelemetry-kube-stack.sh

.PHONY: generate-base-values
generate-base-values:
	@echo "Generating upstream/values.yaml and values-ddot.yaml from values.yaml.tmpl"
	@cd $(REPO_ROOT)/generate && go run .

.PHONY: generate-examples
generate-examples:
	@./render-examples --examples-dir ./upstream/examples --output-name rendered \
		--values ./upstream/values.yaml
	@./render-examples --examples-dir ./examples --output-name rendered-ddot \
		--values ./values-ddot.yaml --values ./dd-agent-kube-stack-values.yaml
