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

type mode struct {
	name string
	// DDOT enables the ddot-flow sections of values.yaml.tmpl.
	DDOT bool
	// Exporter is the exporter that every pipeline in values.yaml.tmpl uses.
	Exporter string
}

var modes = []mode{
	{name: "otlp-http", Exporter: "otlp_http"},
	{name: "ddot", DDOT: true, Exporter: "datadog/exporter"},
}

func render(tmpl *template.Template, m mode) error {
	out, err := os.Create(filepath.Join(guideDir, "values-"+m.name+".yaml"))
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
			fmt.Fprintf(os.Stderr, "failed to render values-%s.yaml: %s\n", m.name, err)
			os.Exit(1)
		}
	}
}
