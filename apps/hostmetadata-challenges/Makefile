# Makefile for the host metadata challenges demo
# Requires: kind, kubectl, docker

CLUSTER_NAME := otel-metadata-demo
NAMESPACE := otel-multi-pipeline-demo
APP_IMAGE := trace-generator:latest

.PHONY: cluster build load deploy all clean logs-gateway logs-agent port-forward generate-traces

# Create a kind cluster
cluster:
	@echo "Creating kind cluster '$(CLUSTER_NAME)'..."
	kind create cluster --name $(CLUSTER_NAME) --wait 60s

# Build the sample app Docker image
build:
	@echo "Building trace-generator image..."
	docker build -t $(APP_IMAGE) ./app

# Load the image into the kind cluster
load: build
	@echo "Loading image into kind cluster..."
	kind load docker-image $(APP_IMAGE) --name $(CLUSTER_NAME)

# Deploy all Kubernetes manifests
deploy:
	@echo "Deploying manifests..."
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/kube-state-metrics.yaml
	kubectl apply -f k8s/otel-agent-configmap.yaml
	kubectl apply -f k8s/otel-agent-daemonset.yaml
	kubectl apply -f k8s/otel-gateway-configmap.yaml
	kubectl apply -f k8s/otel-gateway-deployment.yaml
	kubectl apply -f k8s/app-deployment.yaml
	@echo ""
	@echo "Waiting for pods to be ready..."
	kubectl -n $(NAMESPACE) wait --for=condition=ready pod -l app=kube-state-metrics --timeout=120s
	kubectl -n $(NAMESPACE) wait --for=condition=ready pod -l app=otel-gateway --timeout=120s
	kubectl -n $(NAMESPACE) wait --for=condition=ready pod -l app=trace-generator --timeout=120s
	@echo ""
	@echo "All pods ready. Run 'make logs-gateway' to inspect resource attributes."

# Full setup: cluster → build → load → deploy
all: cluster load deploy

# Show gateway collector logs (where all three resource shapes are visible)
logs-gateway:
	kubectl -n $(NAMESPACE) logs -l app=otel-gateway --tail=200 -f

# Show agent collector logs
logs-agent:
	kubectl -n $(NAMESPACE) logs -l app=otel-agent --tail=100 -f

# Port-forward to the sample app
port-forward:
	kubectl -n $(NAMESPACE) port-forward svc/trace-generator 8080:8080

# Generate traces by hitting the sample app endpoint
generate-traces:
	@echo "Generating traces..."
	@for i in $$(seq 1 10); do \
		curl -s http://localhost:8080/generate; \
		sleep 1; \
	done

# Delete the kind cluster
clean:
	kind delete cluster --name $(CLUSTER_NAME)
