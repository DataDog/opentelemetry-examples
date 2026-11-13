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

func render(tmpl *template.Template, m mode) error {
	out, err := os.Create(filepath.Join(guideDir, string(m), "values.yaml"))
	if err != nil {
		return err
	}
	defer out.Close()

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
