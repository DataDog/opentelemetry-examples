TMP_DIRECTORY = ./tmp
chart_name = opentelemetry-kube-stack
datadog_operator_dir = $(TMP_DIRECTORY)/datadog-operator
platform = $(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m)

.PHONY: generate-examples
generate-examples: generate-otel-kube-stack-examples generate-datadog-operator-examples

.PHONY: generate-otel-kube-stack-examples
generate-otel-kube-stack-examples:
	@echo "Generating all examples into their rendered/ subdirectory"
	@mkdir -p $(TMP_DIRECTORY)
	@# Iterate over all example directories
	@echo "Current folder: $$(pwd)"; \
	EXAMPLES_DIR=./examples; \
	EXAMPLES=$$(find $${EXAMPLES_DIR} -maxdepth 1 -mindepth 1 -type d -exec basename \{\} \;); \
	for example in $${EXAMPLES}; do \
		echo "Generating example: $${example}"; \
		VALUES=$$(find $${EXAMPLES_DIR}/$${example} -name '*values.yaml'); \
		rm -rf "$${EXAMPLES_DIR}/$${example}/rendered"; \
		for value in $${VALUES}; do \
			helm template opentelemetry-kube-stack \
				open-telemetry/opentelemetry-kube-stack  \
				--namespace opentelemetry-operator-system \
				--version "0.21.0" \
				--values ./values.yaml --values $${value} \
				--output-dir "$${EXAMPLES_DIR}/$${example}/rendered"; \
			mv $${EXAMPLES_DIR}/$${example}/rendered/opentelemetry-kube-stack/templates/* "$${EXAMPLES_DIR}/$${example}/rendered"; \
			SUBCHARTS_DIR=$${EXAMPLES_DIR}/$${example}/rendered/opentelemetry-kube-stack/charts; \
			if [ -d "$${SUBCHARTS_DIR}" ]; then \
				SUBCHARTS=$$(find $${SUBCHARTS_DIR} -maxdepth 1 -mindepth 1 -type d -exec basename \{\} \;); \
				for subchart in $${SUBCHARTS}; do \
					mkdir -p "$${EXAMPLES_DIR}/$${example}/rendered/$${subchart}"; \
					mv $${SUBCHARTS_DIR}/$${subchart}/templates/* "$${EXAMPLES_DIR}/$${example}/rendered/$${subchart}"; \
				done; \
				rm -rf $${EXAMPLES_DIR}/$${example}/rendered/opentelemetry-kube-stack; \
			fi; \
		done; \
	done; \
	echo "Normalizing auto-generated certificate values"; \
	if [ "$$(uname)" = "Darwin" ]; then \
		find $${EXAMPLES_DIR} -name '*.yaml' -print0 | xargs -0 sed -i '' -e 's/\(caBundle:\) [A-Za-z0-9+/=]\{44,\}/\1 AUTO_GENERATED/' -e 's/\(tls\.crt:\) [A-Za-z0-9+/=]\{44,\}/\1 AUTO_GENERATED/' -e 's/\(tls\.key:\) [A-Za-z0-9+/=]\{44,\}/\1 AUTO_GENERATED/' -e 's/\(ca\.crt:\) [A-Za-z0-9+/=]\{44,\}/\1 AUTO_GENERATED/'; \
	else \
		find $${EXAMPLES_DIR} -name '*.yaml' -exec sed -i'' \
			-e 's/\(caBundle:\) [A-Za-z0-9+/=]\{44,\}/\1 AUTO_GENERATED/' \
			-e 's/\(tls\.crt:\) [A-Za-z0-9+/=]\{44,\}/\1 AUTO_GENERATED/' \
			-e 's/\(tls\.key:\) [A-Za-z0-9+/=]\{44,\}/\1 AUTO_GENERATED/' \
			-e 's/\(ca\.crt:\) [A-Za-z0-9+/=]\{44,\}/\1 AUTO_GENERATED/' \
			\{\} + ; \
	fi

.PHONY: generate-datadog-operator-examples
generate-datadog-operator-examples:
	@echo "Generating the Datadog Operator example into examples-datadog-agent/"
	@mkdir -p $(TMP_DIRECTORY)
	@if [ ! -d "$(datadog_operator_dir)" ]; then \
		echo "Cloning DataDog/datadog-operator..."; \
		git clone --depth 1 https://github.com/DataDog/datadog-operator.git "$(datadog_operator_dir)"; \
	fi; \
	OPERATOR_RENDER_BIN="$(datadog_operator_dir)/bin/$(platform)/operator-render"; \
	if [ ! -x "$${OPERATOR_RENDER_BIN}" ]; then \
		echo "Building operator-render..."; \
		$(MAKE) -C "$(datadog_operator_dir)" build-renderer; \
	fi; \
	EXAMPLE_DIR=./examples-datadog-agent/default; \
	rm -rf $${EXAMPLE_DIR}; \
	mkdir -p $${EXAMPLE_DIR}/rendered; \
	sed \
		-e "s|<CLUSTER_NAME>|my-k8s-cluster|" \
		-e "s| *# <HOSTNAME_FROM_NODE_NAME>||" \
		-e "s|<DD_SITE>|datadoghq.com|" \
		./datadog-agent.yaml \
		> $${EXAMPLE_DIR}/datadog-agent.yaml; \
	$${OPERATOR_RENDER_BIN} \
		--dda $${EXAMPLE_DIR}/datadog-agent.yaml \
		--output $${EXAMPLE_DIR}/rendered/datadog-agent.yaml