package main

var otelcolVersion = "0.150.1"

var hostEnvs = []string{"", "ec2", "gce", "azure"}

var k8sEnvs = []string{"", "eks"}

var configs = []config{
	{"agent", "otelcol-agent", hostEnvs, nil},
	{"agent-container", "otelcol-agent", []string{""}, map[string]any{"Container": true}},
	{"daemonset", "otelcol-agent", k8sEnvs, map[string]any{"Container": true, "K8s": true, "Kubeletstats": true}},
	{"deployment", "otelcol-deployment-config", k8sEnvs, map[string]any{"Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true}},
	{"helm-values/daemonset", "helm-daemonset", k8sEnvs, map[string]any{"Kubeletstats": true}},
	{"helm-values/deployment", "otelcol-deployment", k8sEnvs, map[string]any{"Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true}},

	{"testing/agent-datadog", "otelcol-agent", []string{""}, map[string]any{"DatadogExporter": true, "Testing": true}},
	{"testing/deployment-datadog", "otelcol-deployment", []string{"", "eks"}, map[string]any{"Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true, "Testing": true}},
	{"testing/daemonset-datadog", "otelcol-agent", []string{""}, map[string]any{"Container": true, "K8s": true, "DatadogExporter": true, "Testing": true}},
	{"testing/otel-demo-datadog", "otel-demo", []string{"", "eks"}, map[string]any{"DatadogExporter": true, "Testing": true, "ExperimentalRuntimeMetrics": true, "MapEquivalentMetrics": true, "Kubeletstats": true}},
	{"testing/otel-demo", "otel-demo", []string{"", "eks"}, map[string]any{"Testing": true, "ExperimentalRuntimeMetrics": true, "MapEquivalentMetrics": true}},
}
