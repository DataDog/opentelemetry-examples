TMP_DIRECTORY = ./tmp
chart_name = opentelemetry-kube-stack
datadog_operator_dir = $(TMP_DIRECTORY)/datadog-operator
platform = $(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m)

REPO_ROOT := $(shell git rev-parse --show-toplevel)
CI_SCRIPTS := $(REPO_ROOT)/.github/workflows/scripts

.PHONY: validate
validate:
	@DUMMY_ENV_FILE=$(CI_SCRIPTS)/dummy-credentials.env $(CI_SCRIPTS)/validate-opentelemetry-kube-stack.sh

# generate regenerates the values files from values.yaml.tmpl, then the examples rendered on top of them. The two
# steps run in sequence, even with make -j, because the examples read the values files. It leaves out
# generate-datadog-operator-examples, which reads ddot/datadog-agent.yaml and needs a datadog-operator checkout.
.PHONY: generate
generate:
	@$(MAKE) --no-print-directory generate-base-values
	@$(MAKE) --no-print-directory generate-examples

# check-doc-snippets checks that the configuration snippets of the guides match their source files. After reviewing
# the snippets whose source changed, pass ARGS=--update-hashes to store the new source hashes.
DOC_SNIPPET_GUIDES = ddot/ADAPT-EXISTING-CONFIG.md
.PHONY: check-doc-snippets
check-doc-snippets:
	@python3 $(CI_SCRIPTS)/check-doc-snippets.py $(ARGS) $(DOC_SNIPPET_GUIDES)

# doc-snippets-for lists the guide snippets that quote FILE, to run before changing it, for example
# make doc-snippets-for FILE=values.yaml.tmpl. The snippets quote the generated ddot/values.yaml instead of the
# template, so FILE=values.yaml.tmpl lists the snippets of ddot/values.yaml.
.PHONY: doc-snippets-for
doc-snippets-for:
	@test -n "$(FILE)" || { echo "Usage: make doc-snippets-for FILE=<file>"; exit 2; }
	@python3 $(CI_SCRIPTS)/check-doc-snippets.py \
		--quoted $(if $(filter values.yaml.tmpl,$(FILE)),ddot/values.yaml,$(FILE)) $(DOC_SNIPPET_GUIDES)

.PHONY: generate-base-values
generate-base-values:
	@echo "Generating upstream/values.yaml and ddot/values.yaml from values.yaml.tmpl"
	@cd $(REPO_ROOT)/generate && go run .

.PHONY: generate-examples
generate-examples:
	@./render-examples --examples-dir ./upstream/examples --output-name rendered \
		--values ./upstream/values.yaml
	@./render-examples --examples-dir ./ddot/examples --output-name rendered \
		--values ./ddot/values.yaml

# operator-render reads the Datadog Operator CRDs from the absolute path of the checkout it was built from
# (runtime.Caller): it's rebuilt when that path changes, for example after moving or renaming a parent folder.
.PHONY: generate-datadog-operator-examples
generate-datadog-operator-examples:
	@echo "Generating the Datadog Operator example into ddot/examples-datadog-agent/"
	@mkdir -p $(TMP_DIRECTORY)
	@if [ ! -d "$(datadog_operator_dir)" ]; then \
		echo "Cloning DataDog/datadog-operator..."; \
		git clone --depth 1 https://github.com/DataDog/datadog-operator.git "$(datadog_operator_dir)"; \
	fi; \
	OPERATOR_RENDER_BIN="$(datadog_operator_dir)/bin/$(platform)/operator-render"; \
	OPERATOR_RENDER_BUILT_FROM="$${OPERATOR_RENDER_BIN}.built-from"; \
	OPERATOR_SRC_DIR="$(abspath $(datadog_operator_dir))"; \
	if [ ! -x "$${OPERATOR_RENDER_BIN}" ] || [ "$$(cat "$${OPERATOR_RENDER_BUILT_FROM}" 2>/dev/null)" != "$${OPERATOR_SRC_DIR}" ]; then \
		echo "Building operator-render..."; \
		$(MAKE) -C "$(datadog_operator_dir)" build-renderer; \
		echo "$${OPERATOR_SRC_DIR}" > "$${OPERATOR_RENDER_BUILT_FROM}"; \
	fi; \
	EXAMPLE_DIR=./ddot/examples-datadog-agent/default; \
	rm -rf $${EXAMPLE_DIR}; \
	mkdir -p $${EXAMPLE_DIR}/rendered; \
	kubectl patch --local \
		-f ./ddot/datadog-agent.yaml \
		--type merge \
		-p '{"spec":{"global":{"clusterName":"my-k8s-cluster","site":"datadoghq.com"}}}' \
		-o yaml \
		> $${EXAMPLE_DIR}/datadog-agent.yaml; \
	$${OPERATOR_RENDER_BIN} \
		--dda $${EXAMPLE_DIR}/datadog-agent.yaml \
		--output $${EXAMPLE_DIR}/rendered/datadog-agent.yaml; \
	echo "Normalizing the auto-generated Cluster Agent token"; \
	sed -e 's/^\(  token:\) [A-Za-z0-9+/=]\{16,\}$$/\1 AUTO_GENERATED/' \
		$${EXAMPLE_DIR}/rendered/datadog-agent.yaml > $${EXAMPLE_DIR}/rendered/datadog-agent.yaml.tmp; \
	mv $${EXAMPLE_DIR}/rendered/datadog-agent.yaml.tmp $${EXAMPLE_DIR}/rendered/datadog-agent.yaml
