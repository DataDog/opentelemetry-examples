package main

var otelcolVersion = "0.152.0"

var hostEnvs = []string{"", "ec2", "gce", "azure"}

var k8sEnvs = []string{"", "eks"}

var configs = []config{
	{"agent", "otelcol-agent", hostEnvs, nil},
	{"agent-container", "otelcol-agent", []string{""}, map[string]any{"Container": true}},
	{"daemonset", "otelcol-agent", k8sEnvs, map[string]any{"Container": true, "K8s": true, "KubeletStats": true}},
	{"k8s-objects-datadog", "otelcol-k8s-objects", k8sEnvs, map[string]any{"Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true}},
	{"helm-values/daemonset", "helm-daemonset", k8sEnvs, map[string]any{"KubeletStats": true}},
	{"helm-values/k8s-objects", "helm-k8s-objects", k8sEnvs, map[string]any{"Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true}},

	{"testing/agent-datadog", "otelcol-agent", []string{""}, map[string]any{"DatadogExporter": true, "Testing": true}},
	{"testing/helm-k8s-objects-datadog", "helm-k8s-objects", []string{"", "eks"}, map[string]any{"Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true, "Testing": true}},
	{"testing/daemonset-datadog", "otelcol-agent", []string{""}, map[string]any{"Container": true, "K8s": true, "DatadogExporter": true, "Testing": true}},
	{"testing/otel-demo-datadog", "otel-demo", []string{"", "eks"}, map[string]any{"DatadogExporter": true, "Testing": true, "ExperimentalRuntimeMetrics": true, "MapEquivalentMetrics": true, "KubeletStats": true}},
	{"testing/otel-demo", "otel-demo", []string{"", "eks"}, map[string]any{"Testing": true, "ExperimentalRuntimeMetrics": true, "MapEquivalentMetrics": true}},
}
