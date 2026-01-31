.PHONY: help build test deploy clean fmt lint

# Variables
IMG ?= tenant-master:latest
DOCKER_REGISTRY ?= docker.io
DOCKER_USERNAME ?= your-username
CONTROLLER_GEN ?= $(shell go env GOPATH)/bin/controller-gen
GO := go

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: all
all: build

.PHONY: build
build: fmt vet generate ## Build the operator binary
	$(GO) build -o bin/manager cmd/main.go

.PHONY: run
run: fmt vet generate ## Run the operator locally
	$(GO) run ./cmd/main.go

.PHONY: docker-build
docker-build: build ## Build container image
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push: docker-build ## Push container image to registry
	docker tag $(IMG) $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(IMG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(IMG)

.PHONY: deploy
deploy: ## Deploy the operator to the cluster
	kubectl apply -f config/crd/tenant_crd.yaml
	kubectl apply -f config/rbac/rbac.yaml
	kubectl apply -f config/webhook/webhook.yaml
	kubectl apply -f config/manager/manager.yaml
	kubectl rollout status deployment/tenant-master -n tenant-system

.PHONY: undeploy
undeploy: ## Remove the operator from the cluster
	kubectl delete -f config/manager/manager.yaml
	kubectl delete -f config/webhook/webhook.yaml
	kubectl delete -f config/rbac/rbac.yaml
	kubectl delete -f config/crd/tenant_crd.yaml

.PHONY: test
test: fmt vet generate ## Run tests
	$(GO) test ./... -v -race -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

.PHONY: fmt
fmt: ## Format code
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: generate
generate: ## Generate code (CRD, deepcopy, etc.)
	@echo "Note: Code generation would typically use controller-gen"
	@echo "Install via: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest"

.PHONY: lint
lint: ## Run linters
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

.PHONY: clean
clean: ## Clean up build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

.PHONY: install-tools
install-tools: ## Install development tools
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: kind-setup
kind-setup: ## Setup a local Kind cluster
	kind create cluster --name tenant-master
	kubectl cluster-info --context kind-tenant-master

.PHONY: kind-delete
kind-delete: ## Delete the Kind cluster
	kind delete cluster --name tenant-master

.PHONY: kind-deploy
kind-deploy: docker-build kind-setup deploy ## Build, setup Kind, and deploy

.PHONY: logs
logs: ## Tail operator logs
	kubectl logs -n tenant-system -f deployment/tenant-master

.PHONY: describe-tenants
describe-tenants: ## List and describe all tenants
	kubectl get tenants
	@echo "---"
	@for tenant in $$(kubectl get tenants -o jsonpath='{.items[*].metadata.name}'); do \
		echo "Tenant: $$tenant"; \
		kubectl describe tenant $$tenant; \
		echo "---"; \
	done

.PHONY: sample-create
sample-create: ## Create sample tenants
	kubectl apply -f config/samples/tenant_examples.yaml

.PHONY: sample-delete
sample-delete: ## Delete sample tenants
	kubectl delete -f config/samples/tenant_examples.yaml

.PHONY: metrics
metrics: ## Port-forward Prometheus metrics
	kubectl port-forward -n tenant-system svc/tenant-master-metrics 8080:8080

.PHONY: webhook-logs
webhook-logs: ## Check webhook logs
	@echo "Webhook logs are part of manager logs"
	kubectl logs -n tenant-system deployment/tenant-master -f | grep webhook
