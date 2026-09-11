TMP_DIRECTORY = ./tmp
chart_name = opentelemetry-kube-stack

.PHONY: generate-examples
generate-examples: generate-otel-kube-stack-examples generate-datadog-agent-examples

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
				--version "0.20.8" \
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

.PHONY: generate-datadog-agent-examples
generate-datadog-agent-examples:
	@echo "Generating all Datadog Agent examples into their rendered/ subdirectory"
	@mkdir -p $(TMP_DIRECTORY)
	@# Iterate over all example directories
	@echo "Current folder: $$(pwd)"; \
	EXAMPLES_DIR=./examples-datadog-agent; \
	EXAMPLES=$$(find $${EXAMPLES_DIR} -maxdepth 1 -mindepth 1 -type d -exec basename \{\} \;); \
	for example in $${EXAMPLES}; do \
		echo "Generating example: $${example}"; \
		VALUES=$$(find $${EXAMPLES_DIR}/$${example} -name '*values.yaml'); \
		rm -rf "$${EXAMPLES_DIR}/$${example}/rendered"; \
		for value in $${VALUES}; do \
			helm template ddagent-kube-stack \
				datadog/datadog  \
				--namespace opentelemetry-operator-system \
				--version "3.240.0" \
				--values ./dd-agent-values.yaml --values $${value} \
				--output-dir "$${EXAMPLES_DIR}/$${example}/rendered"; \
			mv $${EXAMPLES_DIR}/$${example}/rendered/datadog/templates/* "$${EXAMPLES_DIR}/$${example}/rendered"; \
			SUBCHARTS_DIR=$${EXAMPLES_DIR}/$${example}/rendered/datadog/charts; \
			if [ -d "$${SUBCHARTS_DIR}" ]; then \
				SUBCHARTS=$$(find $${SUBCHARTS_DIR} -maxdepth 1 -mindepth 1 -type d -exec basename \{\} \;); \
				for subchart in $${SUBCHARTS}; do \
					mkdir -p "$${EXAMPLES_DIR}/$${example}/rendered/$${subchart}"; \
					mv $${SUBCHARTS_DIR}/$${subchart}/templates/* "$${EXAMPLES_DIR}/$${example}/rendered/$${subchart}"; \
				done; \
			fi; \
			rm -rf $${EXAMPLES_DIR}/$${example}/rendered/datadog; \
		done; \
	done; \
	echo "Normalizing auto-generated values"; \
	if [ "$$(uname)" = "Darwin" ]; then \
		find $${EXAMPLES_DIR} -name '*.yaml' -print0 | xargs -0 sed -i '' -e 's/\(install_id:\) .*/\1 "AUTO_GENERATED"/' -e 's/\(install_time:\) .*/\1 "AUTO_GENERATED"/'; \
	else \
		find $${EXAMPLES_DIR} -name '*.yaml' -exec sed -i'' \
			-e 's/\(install_id:\) .*/\1 "AUTO_GENERATED"/' \
			-e 's/\(install_time:\) .*/\1 "AUTO_GENERATED"/' \
			\{\} + ; \
	fi