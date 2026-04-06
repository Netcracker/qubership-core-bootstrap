.PHONY: help test-unit test-integration test-integration-all test-all \
		test-ratelimit test-controller test-api \
		test-integration-ratelimit test-integration-controller test-integration-api \
		clean start-port-forwards stop-port-forwards force

NAMESPACE := core-1-core
TEST_TIMEOUT := 10m
TEST_ENV := TEST_REDIS_ADDR=localhost:6379 TEST_RATELIMIT_ADDR=localhost:8081

GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m

help:
	@echo "Available targets:"
	@echo ""
	@echo "Unit tests (no dependencies):"
	@echo "  make test-unit			  - Run all unit tests"
	@echo "  make test-ratelimit		 - Run ratelimit package unit tests"
	@echo "  make test-controller		- Run controller package unit tests"
	@echo "  make test-api			   - Run API package unit tests"
	@echo "  make test-metrics		   - Run metrics package unit tests"
	@echo ""
	@echo "Integration tests (require port-forwards):"
	@echo "  make test-integration-ratelimit   - Run ratelimit integration tests"
	@echo "  make test-integration-controller  - Run controller integration tests"
	@echo "  make test-integration-api		 - Run API integration tests"
	@echo "  make test-integration-metrics	 - Run metrics integration tests"
	@echo "  make test-integration-all		 - Run all integration tests"
	@echo ""
	@echo "All tests:"
	@echo "  make test-all			   - Run all tests (unit + integration)"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean				  - Clean test cache"

# ============================================
# Port-forwards management
# ============================================

start-port-forwards:
	@echo "$(BLUE)Starting port-forwards...$(NC)"
	@-pkill -f "kubectl port-forward.*redis" 2>/dev/null || true
	@kubectl port-forward -n $(NAMESPACE) service/redis 6379:6379 > /tmp/redis_pf.log 2>&1 & echo $$! > /tmp/redis_pf.pid
	@sleep 5
	@echo "$(GREEN)✓ Port-forwards started (Redis:6379)$(NC)"

stop-port-forwards:
	@-kill $$(cat /tmp/redis_pf.pid 2>/dev/null) 2>/dev/null || true
	@-rm -f /tmp/redis_pf.pid 2>/dev/null || true
	@echo "$(GREEN)✓ Port-forwards stopped$(NC)"

# ============================================

# ============================================

define run-unit-tests
	@go clean -testcache
	@go test -v -short -timeout $(TEST_TIMEOUT) $(1)
endef

test-unit: force
	@echo "$(BLUE)Running all unit tests...$(NC)"
	$(call run-unit-tests, ./pkg/...)
	@echo "$(GREEN)✓ Unit tests passed!$(NC)"

test-ratelimit: force
	@echo "$(BLUE)Running ratelimit unit tests...$(NC)"
	@go clean -testcache
	@go test -v -short -timeout $(TEST_TIMEOUT) ./pkg/ratelimit/...
	@echo "$(GREEN)✓ Ratelimit unit tests passed!$(NC)"

test-ratelimit-bench: force
	@echo "$(BLUE)Running ratelimit benchmarks...$(NC)"
	@go test -bench=. -benchmem ./pkg/ratelimit/...
	@echo "$(GREEN)✓ Ratelimit benchmarks passed!$(NC)"

test-controller: force
	@echo "$(BLUE)Running controller unit tests...$(NC)"
	$(call run-unit-tests, ./pkg/controller/...)
	@echo "$(GREEN)✓ Controller unit tests passed!$(NC)"

test-api: force
	@echo "$(BLUE)Running API unit tests...$(NC)"
	$(call run-unit-tests, ./pkg/api/...)
	@echo "$(GREEN)✓ API unit tests passed!$(NC)"

test-metrics: force
	@echo "$(BLUE)Running metrics unit tests...$(NC)"
	$(call run-unit-tests, ./pkg/metrics/...)
	@echo "$(GREEN)✓ Metrics unit tests passed!$(NC)"

# ============================================

# ============================================

define run-integration-tests
	@$(MAKE) start-port-forwards
	@echo "$(BLUE)Running $(1) integration tests...$(NC)"
	@go clean -testcache
	@$(TEST_ENV) go test -v -tags=integration -timeout $(TEST_TIMEOUT) -run "TestIntegration" $(2)
	@echo "$(GREEN)✓ $(1) integration tests passed!$(NC)"
	@$(MAKE) stop-port-forwards
endef


test-integration-ratelimit: force
	$(call run-integration-tests, ratelimit, ./pkg/ratelimit/...)

test-integration-controller: force
	$(call run-integration-tests, controller, ./pkg/controller/...)

test-integration-api: force
	$(call run-integration-tests, API, ./pkg/api/...)

test-integration-metrics: force
	$(call run-integration-tests, metrics, ./pkg/metrics/...)


test-integration-all: force
	$(call run-integration-tests, all, ./...)


# ============================================
# unit + integration
# ============================================

test-all: force test-unit test-integration-all
	@echo "$(GREEN)✓ All tests (unit + integration) passed!$(NC)"

# ============================================
# Utilities
# ============================================

force:

clean: force
	@echo "$(BLUE)Cleaning...$(NC)"
	@go clean -testcache
	@go mod tidy
	@$(MAKE) stop-port-forwards
	@echo "$(GREEN)✓ Clean completed!$(NC)"

# ============================================
# Local E2E Tests
# ============================================

.PHONY: test-e2e test-integration-e2e

test-e2e: force
	@echo "$(BLUE)Running E2E tests...$(NC)"
	@go clean -testcache
	@go test -v -tags=e2e -timeout $(TEST_TIMEOUT) ./tests/e2e/...
	@echo "$(GREEN)✓ E2E tests passed!$(NC)"

test-integration-e2e: force start-port-forwards
	@echo "$(BLUE)Running E2E integration tests with port-forwards...$(NC)"
	@go clean -testcache
	@TEST_REDIS_ADDR=localhost:6379 TEST_RATELIMIT_ADDR=localhost:8081 \
		go test -v -tags=e2e -timeout $(TEST_TIMEOUT) ./tests/e2e/...
	@$(MAKE) stop-port-forwards
	@echo "$(GREEN)✓ E2E integration tests passed!$(NC)"

# ============================================
# Cloud E2E Tests
# ============================================

cloud-test-port-forward:
	@echo "$(BLUE)Starting port-forwards for cloud tests...$(NC)"
	@-pkill -f "kubectl port-forward" 2>/dev/null || true
	@kubectl port-forward -n $(NAMESPACE) svc/public-gateway-istio 8080:8080 > /tmp/gateway_pf.log 2>&1 & echo $$! > /tmp/gateway_pf.pid
	@kubectl port-forward -n $(NAMESPACE) svc/ratelimit-service 8082:8082 > /tmp/operator_pf.log 2>&1 & echo $$! > /tmp/operator_pf.pid
	@kubectl port-forward -n $(NAMESPACE) svc/redis 6379:6379 > /tmp/redis_pf.log 2>&1 & echo $$! > /tmp/redis_pf.pid
	@sleep 5
	@echo "$(GREEN)✓ Port-forwards started (Gateway:8080, Operator:8082, Redis:6379)$(NC)"

stop-cloud-test-port-forward:
	@echo "$(BLUE)Stopping port-forwards...$(NC)"
	@-kill $$(cat /tmp/gateway_pf.pid 2>/dev/null) 2>/dev/null || true
	@-kill $$(cat /tmp/operator_pf.pid 2>/dev/null) 2>/dev/null || true
	@-kill $$(cat /tmp/redis_pf.pid 2>/dev/null) 2>/dev/null || true
	@-rm -f /tmp/gateway_pf.pid /tmp/operator_pf.pid /tmp/redis_pf.pid
	@echo "$(GREEN)✓ Port-forwards stopped$(NC)"


test-cloud-e2e: cloud-test-port-forward
	@echo "$(BLUE)Running cloud E2E tests...$(NC)"
	@go clean -testcache
	@TEST_JWT_TOKEN=$(TEST_JWT_TOKEN) \
		go test -v -tags=cloud_e2e -timeout $(TEST_TIMEOUT) ./tests/cloud-e2e/...
	@$(MAKE) stop-cloud-test-port-forward
	@echo "$(GREEN)✓ Cloud E2E tests passed$(NC)"

test-cloud-e2e-performance: cloud-test-port-forward
	@echo "$(BLUE)Running cloud E2E performance tests...$(NC)"
	@go clean -testcache
	@TEST_JWT_TOKEN=$(TEST_JWT_TOKEN) \
		go test -v -tags=cloud_e2e -run TestCloudE2E_Performance -timeout $(TEST_TIMEOUT) ./tests/cloud-e2e/...
	@$(MAKE) stop-cloud-test-port-forward
	@echo "$(GREEN)✓ Performance tests passed$(NC)"