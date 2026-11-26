// Command generate renders the opentelemetry-kube-stack guide's
// upstream/values.yaml and ddot/values.yaml files from its values.yaml.tmpl,
// one per exporter mode.
// Run via `make generate-base-values` in
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

type mode string

const (
	upstream mode = "upstream"
	ddot     mode = "ddot"
)

var modes = []mode{upstream, ddot}

// DDOT enables the ddot-flow sections of values.yaml.tmpl.
func (m mode) DDOT() bool {
	return m == ddot
}

// Exporter is the exporter that every pipeline in values.yaml.tmpl uses.
func (m mode) Exporter() string {
	if m.DDOT() {
		return "datadog/exporter"
	}
	return "otlp_http"
}

// header tells readers of the generated files to edit values.yaml.tmpl instead.
const header = `# Generated from ../values.yaml.tmpl by "make generate" in the parent folder: don't edit this file.
# Edit values.yaml.tmpl, then run "make generate" to regenerate upstream/values.yaml, ddot/values.yaml, and the
# rendered examples.
`

func render(tmpl *template.Template, m mode) error {
	out, err := os.Create(filepath.Join(guideDir, string(m), "values.yaml"))
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.WriteString(header); err != nil {
		return err
	}
	return tmpl.Execute(out, m)
}

func main() {
	tmpl := template.Must(template.ParseFiles(filepath.Join(guideDir, "values.yaml.tmpl")))
	for _, m := range modes {
		if err := render(tmpl, m); err != nil {
			fmt.Fprintf(os.Stderr, "failed to render %s/values.yaml: %s\n", m, err)
			os.Exit(1)
		}
	}
}
