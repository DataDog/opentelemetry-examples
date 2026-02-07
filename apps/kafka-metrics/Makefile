.PHONY: help build-jmxreceiver up-jmxreceiver down-jmxreceiver build-gatherer up-gatherer down-gatherer clean

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

# JMX Receiver mode - collector uses built-in JMX receiver
build-jmxreceiver: ## Build images for JMX Receiver mode
	docker compose -f docker-compose.jmxreceiver.yaml build

up-jmxreceiver: ## Start services in JMX Receiver mode
	docker compose -f docker-compose.jmxreceiver.yaml up

down-jmxreceiver: ## Stop services in JMX Receiver mode
	docker compose -f docker-compose.jmxreceiver.yaml down

# JMX Metrics Gatherer mode - uses sidecar JMX Metrics Gatherer containers
build-gatherer: ## Build images for JMX Metrics Gatherer mode
	docker compose -f docker-compose.jmxmetricsgatherer.yaml build

up-gatherer: ## Start services in JMX Metrics Gatherer mode
	docker compose -f docker-compose.jmxmetricsgatherer.yaml up

down-gatherer: ## Stop services in JMX Metrics Gatherer mode
	docker compose -f docker-compose.jmxmetricsgatherer.yaml down

clean: ## Remove all containers, images, and volumes
	docker compose -f docker-compose.jmxreceiver.yaml down --rmi all --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.jmxmetricsgatherer.yaml down --rmi all --volumes --remove-orphans 2>/dev/null || true
