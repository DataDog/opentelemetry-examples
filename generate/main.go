// Command generate renders the opentelemetry-kube-stack guide's
// upstream/values.yaml and values-ddot.yaml files from its values.yaml.tmpl,
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

func render(tmpl *template.Template, m mode, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	return tmpl.Execute(out, m)
}

func main() {
	tmpl := template.Must(template.ParseFiles(filepath.Join(guideDir, "values.yaml.tmpl")))
	for _, m := range modes {
		path := filepath.Join(guideDir, string(m), "values.yaml")
		if m == ddot {
			path = filepath.Join(guideDir, "values-ddot.yaml")
		}
		if err := render(tmpl, m, path); err != nil {
			fmt.Fprintf(os.Stderr, "failed to render %s: %s\n", path, err)
			os.Exit(1)
		}
	}
}
