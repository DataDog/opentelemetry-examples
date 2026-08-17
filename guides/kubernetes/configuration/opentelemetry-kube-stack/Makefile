TMP_DIRECTORY = ./tmp
chart_name = opentelemetry-kube-stack

.PHONY: generate-examples
generate-examples:
	@echo "Generating all examples into their rendered/ subdirectory"
	@mkdir -p $(TMP_DIRECTORY)
	@# Iterate over all example directories
	@echo "Current folder: $$(pwd)"; \
	EXAMPLES_DIR=./examples; \
	EXAMPLES=$$(find $${EXAMPLES_DIR} -maxdepth 1 -mindepth 1 -type d -exec basename \{\} \;); \
	for example in $${EXAMPLES}; do \
		echo "Generating example: $${example}"; \
		VALUES=$$(find $${EXAMPLES_DIR}/$${example} -name *values.yaml); \
		rm -rf "$${EXAMPLES_DIR}/$${example}/rendered"; \
		for value in $${VALUES}; do \
			helm template opentelemetry-kube-stack \
				/Users/cyrille.leclerc/src/opentelemetry/opentelemetry-helm-charts/charts/opentelemetry-kube-stack  \
				--namespace opentelemetry-operator-system \
				--version "0.20.1" \
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