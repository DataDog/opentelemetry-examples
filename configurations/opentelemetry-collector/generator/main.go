package main

import (
	"bytes"
	"flag"
	"fmt"
	"maps"
	"os"
	"path"
	"strings"
	"text/template"
)

type config struct {
	name      string
	template  string
	cloudEnvs []string
	vars      map[string]any
}

func renderTemplate(templates *template.Template, outputsPath string, name string, template string, cloudEnv string, vars map[string]any) error {
	outputName := name
	if cloudEnv != "" {
		outputName += "-" + cloudEnv
	}
	output, err := os.Create(path.Join(outputsPath, outputName+".yaml"))
	if err != nil {
		return err
	}
	if vars == nil {
		vars = make(map[string]any)
	} else {
		vars = maps.Clone(vars)
	}
	vars["CloudEnvironment"] = cloudEnv
	vars["Preamble"] = true
	vars["OtelcolVersion"] = otelcolVersion
	return templates.ExecuteTemplate(output, template+".tmpl", vars)
}

func renderConfigs(sourcesPath string, outputsPath string) error {
	templates := template.New("root")
	templates = templates.Funcs(template.FuncMap{
		"include": func(name string, data any) (string, error) {
			buf := bytes.NewBuffer(nil)
			if err := templates.ExecuteTemplate(buf, name+".tmpl", data); err != nil {
				return "", err
			}
			return strings.TrimSpace(buf.String()), nil
		},
		"indent": func(spaces int, text string) string {
			indent := strings.Repeat(" ", spaces)
			return strings.ReplaceAll(text, "\n", "\n"+indent)
		},
		"errorf": func(msg string, args ...any) (string, error) {
			return "", fmt.Errorf(msg, args...)
		},
		"set": func(data any, key string, value any) string {
			data.(map[string]any)[key] = value
			return ""
		},
	})
	var err error
	templates, err = templates.ParseGlob(path.Join(sourcesPath, "*.tmpl"))
	if err == nil {
		templates, err = templates.ParseGlob(path.Join(sourcesPath, "**/*.tmpl"))
	}
	if err != nil {
		return err
	}

	for _, config := range configs {
		for _, cloudEnv := range config.cloudEnvs {
			if err := renderTemplate(templates, outputsPath, config.name, config.template, cloudEnv, config.vars); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: generator [flags] [templates directory] [output directory]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(2)
	}
	sourcesPath := args[0]
	outputPath := args[1]

	if err := renderConfigs(sourcesPath, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render config files: %s\n", err)
		os.Exit(1)
	}
}
