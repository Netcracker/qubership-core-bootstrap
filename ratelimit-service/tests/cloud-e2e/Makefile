# Makefile

.PHONY: help build docker-build docker-push deploy undeploy test-cloud-e2e test-cloud-e2e-performance clean

NAMESPACE := core-1-core
HELM_RELEASE := ratelimit
DOCKER_REGISTRY ?= ratelimit
DOCKER_TAG ?= latest
TEST_TIMEOUT := 10m

GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m

help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Deploy:"
	@echo "  make build                    - Build Go binary"
	@echo "  make docker-build             - Build Docker image"
	@echo "  make docker-push              - Push Docker image to registry"
	@echo "  make deploy                   - Deploy to cloud using Helm"
	@echo "  make undeploy                 - Uninstall from cloud"
	@echo "  make redeploy                 - Redeploy (undeploy + deploy)"
	@echo ""
	@echo "Testing:"
	@echo "  make test-cloud-e2e           - Run cloud E2E tests"
	@echo "  make test-cloud-e2e-performance - Run performance tests"
	@echo ""
	@echo "Local development:"
	@echo "  make test-unit                - Run unit tests"
	@echo "  make test-integration         - Run integration tests"
	@echo "  make test-all                 - Run all tests"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean                    - Clean build artifacts"
	@echo "  make port-forward             - Start port-forwards for testing"

# ============================================
# Build
# ============================================

build:
	@echo "$(BLUE)Building Go binary...$(NC)"
	@go build -o bin/ratelimit cmd/operator/main.go
	@echo "$(GREEN)✓ Build complete$(NC)"

docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_REGISTRY):$(DOCKER_TAG) .
	@echo "$(GREEN)✓ Docker image built: $(DOCKER_REGISTRY):$(DOCKER_TAG)$(NC)"

docker-push:
	@echo "$(BLUE)Pushing Docker image...$(NC)"
	@docker push $(DOCKER_REGISTRY):$(DOCKER_TAG)
	@echo "$(GREEN)✓ Docker image pushed$(NC)"

# ============================================
# Helm templates
# ============================================

helm-template:
	@echo "$(BLUE)Generating Helm templates...$(NC)"
	@helm template $(HELM_RELEASE) ./helm/ratelimit --namespace $(NAMESPACE)
	@echo "$(GREEN)✓ Templates generated$(NC)"

helm-lint:
	@echo "$(BLUE)Linting Helm chart...$(NC)"
	@helm lint ./helm/ratelimit
	@echo "$(GREEN)✓ Helm chart is valid$(NC)"

# ============================================
# Deploy / Undeploy
# ============================================

deploy: docker-build
	@echo "$(BLUE)Deploying to cloud...$(NC)"
	@kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install $(HELM_RELEASE) ./helm/ratelimit \
		--namespace $(NAMESPACE) \
		--set image.repository=$(DOCKER_REGISTRY) \
		--set image.tag=$(DOCKER_TAG) \
		--wait \
		--timeout 5m
	@echo "$(GREEN)✓ Deployment complete$(NC)"
	@$(MAKE) wait-for-ready

undeploy:
	@echo "$(BLUE)Uninstalling from cloud...$(NC)"
	@helm uninstall $(HELM_RELEASE) --namespace $(NAMESPACE) || true
	@echo "$(GREEN)✓ Uninstall complete$(NC)"

redeploy: undeploy deploy
	@echo "$(GREEN)✓ Redeploy complete$(NC)"

wait-for-ready:
	@echo "$(BLUE)Waiting for operator to be ready...$(NC)"
	@kubectl wait --for=condition=ready pod -l app=ratelimit -n $(NAMESPACE) --timeout=120s
	@echo "$(GREEN)✓ Operator is ready$(NC)"

# ============================================
# Port-forwards
# ============================================

port-forward:
	@echo "$(BLUE)Starting port-forwards...$(NC)"
	@-pkill -f "kubectl port-forward.*public-gateway" 2>/dev/null || true
	@-pkill -f "kubectl port-forward.*ratelimit" 2>/dev/null || true
	@-pkill -f "kubectl port-forward.*redis" 2>/dev/null || true
	@kubectl port-forward -n $(NAMESPACE) service/public-gateway-istio 8080:8080 > /tmp/gateway_pf.log 2>&1 & echo $$! > /tmp/gateway_pf.pid
	@kubectl port-forward -n $(NAMESPACE) service/ratelimit-service 8082:8082 > /tmp/operator_pf.log 2>&1 & echo $$! > /tmp/operator_pf.pid
	@kubectl port-forward -n $(NAMESPACE) service/redis 6379:6379 > /tmp/redis_pf.log 2>&1 & echo $$! > /tmp/redis_pf.pid
	@sleep 5
	@echo "$(GREEN)✓ Port-forwards started (Gateway:8080, Operator:8082, Redis:6379)$(NC)"

stop-port-forward:
	@echo "$(BLUE)Stopping port-forwards...$(NC)"
	@-kill $$(cat /tmp/gateway_pf.pid 2>/dev/null) 2>/dev/null || true
	@-kill $$(cat /tmp/operator_pf.pid 2>/dev/null) 2>/dev/null || true
	@-kill $$(cat /tmp/redis_pf.pid 2>/dev/null) 2>/dev/null || true
	@-rm -f /tmp/gateway_pf.pid /tmp/operator_pf.pid /tmp/redis_pf.pid
	@echo "$(GREEN)✓ Port-forwards stopped$(NC)"

# ============================================
# Cloud E2E Tests
# ============================================

test-cloud-e2e: port-forward
	@echo "$(BLUE)Running cloud E2E tests...$(NC)"
	@go clean -testcache
	@TEST_JWT_TOKEN=$(TEST_JWT_TOKEN) \
		go test -v -tags=cloud_e2e -timeout $(TEST_TIMEOUT) ./test/cloud-e2e/...
	@$(MAKE) stop-port-forward
	@echo "$(GREEN)✓ Cloud E2E tests passed$(NC)"

test-cloud-e2e-performance: port-forward
	@echo "$(BLUE)Running cloud E2E performance tests...$(NC)"
	@go clean -testcache
	@TEST_JWT_TOKEN=$(TEST_JWT_TOKEN) \
		go test -v -tags=cloud_e2e -run TestCloudE2E_Performance -timeout $(TEST_TIMEOUT) ./test/cloud-e2e/...
	@$(MAKE) stop-port-forward
	@echo "$(GREEN)✓ Performance tests passed$(NC)"

# ============================================
# Local Tests
# ============================================

test-unit:
	@echo "$(BLUE)Running unit tests...$(NC)"
	@go clean -testcache
	@go test -v -short -timeout $(TEST_TIMEOUT) ./pkg/...
	@echo "$(GREEN)✓ Unit tests passed$(NC)"

test-integration: start-port-forwards
	@echo "$(BLUE)Running integration tests...$(NC)"
	@go clean -testcache
	@TEST_REDIS_ADDR=localhost:6379 \
		go test -v -tags=integration -timeout $(TEST_TIMEOUT) ./pkg/...
	@$(MAKE) stop-port-forwards
	@echo "$(GREEN)✓ Integration tests passed$(NC)"

test-all: test-unit test-integration
	@echo "$(GREEN)✓ All tests passed$(NC)"

# ============================================
# Utilities
# ============================================

clean:
	@echo "$(BLUE)Cleaning...$(NC)"
	@rm -rf bin/
	@go clean -testcache
	@$(MAKE) stop-port-forward
	@echo "$(GREEN)✓ Clean complete$(NC)"

.PHONY: start-port-forwards stop-port-forwards
start-port-forwards:
	@echo "$(BLUE)Starting port-forwards for local tests...$(NC)"
	@-pkill -f "kubectl port-forward.*redis" 2>/dev/null || true
	@kubectl port-forward -n $(NAMESPACE) service/redis 6379:6379 > /tmp/redis_pf.log 2>&1 & echo $$! > /tmp/redis_pf.pid
	@sleep 3
	@echo "$(GREEN)✓ Port-forwards started$(NC)"

stop-port-forwards:
	@-kill $$(cat /tmp/redis_pf.pid 2>/dev/null) 2>/dev/null || true
	@-rm -f /tmp/redis_pf.pid 2>/dev/null || true