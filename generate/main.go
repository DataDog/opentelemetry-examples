// Command generate renders the opentelemetry-kube-stack guide's
// upstream/values.yaml and values-ddot.yaml files from its values.yaml.tmpl,
// one per exporter mode, and ddot/values.yaml for the standalone ddot
// installer.
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

// Paths are the files that the mode's values are written to.
func (m mode) Paths() []string {
	if m == otlpHTTP {
		return []string{filepath.Join(guideDir, "upstream", "values.yaml")}
	}
	return []string{
		filepath.Join(guideDir, "values-"+string(m)+".yaml"),
		filepath.Join(guideDir, "ddot", "values.yaml"),
	}
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
		for _, path := range m.Paths() {
			if err := render(tmpl, m, path); err != nil {
				fmt.Fprintf(os.Stderr, "failed to render %s: %s\n", path, err)
				os.Exit(1)
			}
		}
	}
}
