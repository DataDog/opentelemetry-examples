package main

var otelcolVersion = "0.150.1"

var hostEnvs = []string{"", "ec2", "gce", "azure"}

var k8sEnvs = []string{"", "eks"}

var configs = []config{
	{"otelcol-agent", "otelcol-agent", hostEnvs, nil},
	{"otelcol-agent-container", "otelcol-agent", []string{""}, map[string]any{"Container": true}},
	{"otelcol-daemonset", "otelcol-agent", k8sEnvs, map[string]any{"Container": true, "K8s": true}},
	{"helm-daemonset", "helm-daemonset", k8sEnvs, nil},
	{"otel-demo", "otel-demo", k8sEnvs, nil},
	{"otel-demo-datadog", "otel-demo", []string{"eks"}, map[string]any{"DatadogExporter": true}},
}
