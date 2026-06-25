package main

var otelcolVersion = "0.152.0"

var hostEnvs = []string{"", "ec2", "gce", "azure"}

var k8sDaemonsetEnvs = []string{"", "eks", "gke", "gke-autopilot", "aks", "aks-automatic"}
var k8sDeploymentEnvs = []string{"", "eks", "gke", "aks"}

var configs = []config{
	{"agent", "otelcol-agent", hostEnvs, nil},
	{"agent-container", "otelcol-agent", []string{""}, map[string]any{"Container": true}},
	{"daemonset", "otelcol-agent", k8sDaemonsetEnvs, map[string]any{"Container": true, "K8s": true, "KubeletStats": true}},
	{"k8s-objects-datadog", "otelcol-k8s-objects", k8sDeploymentEnvs, map[string]any{"Container": true, "K8s": true, "Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true}},
	{"helm-values/daemonset", "helm-daemonset", k8sDaemonsetEnvs, map[string]any{"KubeletStats": true}},
	{"helm-values/k8s-objects-datadog", "helm-k8s-objects", k8sDeploymentEnvs, map[string]any{"Container": true, "K8s": true, "Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true}},

	{"testing/agent-datadog", "otelcol-agent", []string{""}, map[string]any{"DatadogExporter": true, "Testing": true}},
	{"testing/helm-k8s-objects-datadog", "helm-k8s-objects", []string{"", "eks"}, map[string]any{"Container": true, "K8s": true, "Deployment": true, "DatadogExporter": true, "KSM": true, "K8sObjects": true, "Testing": true}},
	{"testing/daemonset-datadog", "otelcol-agent", []string{"", "aks-automatic"}, map[string]any{"Container": true, "K8s": true, "DatadogExporter": true, "Testing": true}},
	{"testing/otel-demo-datadog", "otel-demo", []string{"", "eks"}, map[string]any{"DatadogExporter": true, "Testing": true, "ExperimentalRuntimeMetrics": true, "MapEquivalentMetrics": true, "KubeletStats": true}},
	{"testing/otel-demo", "otel-demo", []string{"", "eks"}, map[string]any{"Testing": true, "ExperimentalRuntimeMetrics": true, "MapEquivalentMetrics": true, "DualShip": true}},
}
