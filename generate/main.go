// Command generate renders the opentelemetry-kube-stack guide's
// values-<mode>.yaml files from its values.yaml.tmpl, one per exporter mode.
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
	otlpHTTP mode = "otlp-http"
	ddot     mode = "ddot"
)

var modes = []mode{otlpHTTP, ddot}

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
	out, err := os.Create(filepath.Join(guideDir, "values-"+string(m)+".yaml"))
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
			fmt.Fprintf(os.Stderr, "failed to render values-%s.yaml: %s\n", m, err)
			os.Exit(1)
		}
	}
}
