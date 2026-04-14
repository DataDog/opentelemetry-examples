package main

var otelcolVersion = "0.145.0"

var hostEnvs = []string{"", "ec2", "gce"}

var k8sEnvs = []string{"", "eks"}

var configs = []config{
	{"otelcol-host", "otelcol-agent", hostEnvs, nil},
	{"otelcol-daemonset", "otelcol-agent", k8sEnvs, map[string]any{"K8s": true}},
	{"helm-daemonset", "helm-daemonset", k8sEnvs, nil},
	{"otel-demo", "otel-demo", k8sEnvs, nil},
	{"otel-demo-datadog", "otel-demo", []string{"eks"}, map[string]any{"DatadogExporter": true}},
}
