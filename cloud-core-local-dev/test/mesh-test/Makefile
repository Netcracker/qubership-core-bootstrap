# =============================================================================
# MESH TESTING TARGETS
# =============================================================================

# Define colors for output
YELLOW := \033[1;33m
GREEN := \033[1;32m
RED := \033[0;31m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m

# Phony targets
.PHONY: help validate-input mesh-smoke-test mesh-smoke-test-print-start deploy-mesh-test wait-for-mesh-test-pod run-mesh-test-curl cleanup-mesh-test

MESH_TEST_YAML_FILE ?= mesh-test.yaml
MESH_TEST_SCRIPT_FILE ?= mesh-test.sh
MESH_TEST_SCRIPT := $(shell cat $(MESH_TEST_SCRIPT_FILE))

# Help target
help:
	@echo "$(BLUE)=== Mesh Smoke Test Makefile ===$(NC)"
	@echo ""
	@echo "Usage:"
	@echo "  make <target> [NAMESPACE=ns]"
	@echo ""
	@echo "Targets:"
	@echo "  mesh-smoke-test         - Run full mesh smoke test flow"
	@echo "    mesh-smoke-test-print-start - Print start message"
	@echo "    validate-input        - Validate required parameters"
	@echo "    deploy-mesh-test       - Apply mesh test resources"
	@echo "    wait-for-mesh-test-pod - Wait for mesh test pod to be ready"
	@echo "    run-mesh-test-curl     - Run curl-based mesh test"
	@echo "    cleanup-mesh-test      - Delete mesh test resources"
	@echo ""
	@echo "Variables:"
	@echo "    NAMESPACE              - Namespace where the mesh test runs (required)"
	@echo ""

# Validate input parameters
validate-input:
	@echo "$(BLUE)=== Validating input parameters...$(NC)"
	@if [ -z "$(NAMESPACE)" ]; then \
		echo "$(RED)Error: NAMESPACE is empty. Set NAMESPACE for smoke test.$(NC)"; \
		exit 1; \
	fi;
	@echo "$(CYAN)--- NAMESPACE: $(NAMESPACE)$(NC)"
	@echo "$(GREEN)=== Input parameters validated successfully$(NC)"
	@echo ""

# Main mesh smoke test target
mesh-smoke-test: mesh-smoke-test-print-start validate-input deploy-mesh-test wait-for-mesh-test-pod run-mesh-test-curl cleanup-mesh-test
	@echo "$(GREEN)=== Cloud Core Mesh Smoke Test completed$(NC)"
	@echo ""

# Print start message for mesh smoke test
mesh-smoke-test-print-start:
	@echo "$(BLUE)=== Starting Cloud Core Mesh Smoke Test for namespace \"$(NAMESPACE)\"...$(NC)"
	@echo ""

# Deploy mesh test resources
deploy-mesh-test:
	@echo "$(YELLOW)--- Deploying mesh test resources...$(NC)"
	@if [ -f "$(MESH_TEST_YAML_FILE)" ]; then \
		kubectl apply -f $(MESH_TEST_YAML_FILE) -n $(NAMESPACE); \
		echo "$(GREEN)--- Mesh test resources deployed successfully$(NC)"; \
	else \
		echo "$(RED)Error: $(MESH_TEST_YAML_FILE) not found$(NC)"; \
		echo "$(CYAN)Please create $(MESH_TEST_YAML_FILE) file with your test resources$(NC)"; \
		exit 1; \
	fi
	@echo ""

# Wait for mesh test pod to be ready
wait-for-mesh-test-pod:
	@echo "$(YELLOW)--- Waiting for mesh test pod to be ready (timeout: 3 minutes)...$(NC)"
	@TIMEOUT=180; \
	START_TIME=$$(date +%s); \
	while true; do \
		CURRENT_TIME=$$(date +%s); \
		ELAPSED_TIME=$$((CURRENT_TIME - START_TIME)); \
		if [ $$ELAPSED_TIME -ge $$TIMEOUT ]; then \
			echo "$(RED)Timeout reached after 3 minutes. Mesh test pod is not ready.$(NC)"; \
			echo "$(CYAN)Checking pod status:$(NC)"; \
			kubectl get pods -n $(NAMESPACE) --selector=app=mesh-test -o wide; \
			echo "$(CYAN)Pod logs:$(NC)"; \
			kubectl logs -n $(NAMESPACE) --selector=app=mesh-test --tail=50; \
			exit 1; \
		fi; \
		READY_PODS=$$(kubectl get pods -n $(NAMESPACE) --selector=app=mesh-test --field-selector=status.phase=Running -o jsonpath='{.items[?(@.status.containerStatuses[0].ready==true)].metadata.name}' | wc -w); \
		if [ "$$READY_PODS" -ge 1 ]; then \
			echo "$(GREEN)Mesh test pod is ready! ($$READY_PODS ready pods)$(NC)"; \
			break; \
		fi; \
		echo "Waiting for mesh test pod to be ready... ($$READY_PODS ready) - $$((TIMEOUT - ELAPSED_TIME))s remaining"; \
		sleep 5; \
	done
	@echo ""

# Run mesh test through public gateway using curl
run-mesh-test-curl:
	@echo "$(YELLOW)--- Running mesh test through public gateway...$(NC)"
	@echo "$(CYAN)--- Creating temporary curl pod for testing...$(NC)"
	@kubectl run mesh-test-curl-$$(date +%s) --image=curlimages/curl:latest -n $(NAMESPACE) --rm -i --restart=Never -- \
		sh -c "$(MESH_TEST_SCRIPT)" || (echo "$(RED)✗ Mesh smoke test failed$(NC)" && exit 1)
	@echo "$(GREEN)=== Mesh smoke test completed successfully for namespace \"$(NAMESPACE)\"(NC)"
	@echo ""

# Clean up mesh test resources
cleanup-mesh-test: validate-input
	@echo "$(YELLOW)--- Cleaning up mesh test resources...$(NC)"
	@if [ -f "$(MESH_TEST_YAML_FILE)" ]; then \
		echo "$(CYAN)--- Deleting mesh test resources from $(NAMESPACE) namespace...$(NC)"; \
		kubectl delete -f $(MESH_TEST_YAML_FILE) -n $(NAMESPACE) --ignore-not-found=true || true; \
		echo "$(GREEN)--- Mesh test resources cleaned up successfully$(NC)"; \
	fi
	@echo ""
