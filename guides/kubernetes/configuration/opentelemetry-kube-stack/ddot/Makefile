TMP_DIRECTORY = ./tmp
chart_name = opentelemetry-kube-stack
datadog_operator_dir = $(TMP_DIRECTORY)/datadog-operator
platform = $(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m)

.PHONY: generate-examples
generate-examples: generate-otel-kube-stack-examples generate-datadog-operator-examples

.PHONY: generate-otel-kube-stack-examples
generate-otel-kube-stack-examples:
	@../render-examples --examples-dir ./examples --output-name rendered --values ./values.yaml

# operator-render reads the Datadog Operator CRDs from the absolute path of the checkout it was built from
# (runtime.Caller): it's rebuilt when that path changes, for example after moving or renaming a parent folder.
.PHONY: generate-datadog-operator-examples
generate-datadog-operator-examples:
	@echo "Generating the Datadog Operator example into examples-datadog-agent/"
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
	EXAMPLE_DIR=./examples-datadog-agent/default; \
	rm -rf $${EXAMPLE_DIR}; \
	mkdir -p $${EXAMPLE_DIR}/rendered; \
	sed \
		-e "s|<CLUSTER_NAME>|my-k8s-cluster|" \
		-e "/<HOSTNAME_FROM_NODE_NAME:/d" \
		-e "s|<DD_SITE>|datadoghq.com|" \
		./datadog-agent.yaml \
		> $${EXAMPLE_DIR}/datadog-agent.yaml; \
	$${OPERATOR_RENDER_BIN} \
		--dda $${EXAMPLE_DIR}/datadog-agent.yaml \
		--output $${EXAMPLE_DIR}/rendered/datadog-agent.yaml; \
	echo "Normalizing the auto-generated Cluster Agent token"; \
	sed -e 's/^\(  token:\) [A-Za-z0-9+/=]\{16,\}$$/\1 AUTO_GENERATED/' \
		$${EXAMPLE_DIR}/rendered/datadog-agent.yaml > $${EXAMPLE_DIR}/rendered/datadog-agent.yaml.tmp; \
	mv $${EXAMPLE_DIR}/rendered/datadog-agent.yaml.tmp $${EXAMPLE_DIR}/rendered/datadog-agent.yaml