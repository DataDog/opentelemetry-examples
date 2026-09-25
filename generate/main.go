// Command generate renders the opentelemetry-kube-stack guide's values.yaml
// from its values.yaml.tmpl. Run via `make generate-base-values` in
// guides/kubernetes/configuration/opentelemetry-kube-stack (cds into this
// directory first).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const guideDir = "../guides/kubernetes/configuration/opentelemetry-kube-stack"

func main() {
	tmpl := template.Must(template.ParseFiles(filepath.Join(guideDir, "values.yaml.tmpl")))

	out, err := os.Create(filepath.Join(guideDir, "values.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create values.yaml: %s\n", err)
		os.Exit(1)
	}
	defer out.Close()

	if err := tmpl.Execute(out, nil); err != nil {
		fmt.Fprintf(os.Stderr, "failed to render values.yaml: %s\n", err)
		os.Exit(1)
	}
}
