 # =============================================================================
 # ISTIO MESH TESTS
 # =============================================================================

# Define colors for output
YELLOW := \033[1;33m
GREEN := \033[1;32m
RED := \033[0;31m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m

 .PHONY: help validate-input check-gateway-services check-ztunnel-logs run-mesh-test check-public-gateway-istio-logs check-waypoint-logs test-start-message test collect-logs
 
 NAMESPACE ?= 
 ISTIO_NAMESPACE ?= istio-system
 LOG_TAIL_LINES ?= 1000
 LOG_OUTPUT_DIR ?= ./istio-logs
 LOG_OUTPUT_PATH = $(LOG_OUTPUT_DIR)/$(NAMESPACE)

 PUBLIC_GATEWAY_SERVICE = public-gateway-service
 PUBLIC_GATEWAY_DEPLOYMENT = public-gateway-istio
 
 PRIVATE_GATEWAY_SERVICE = private-gateway-service
 PRIVATE_GATEWAY_DEPLOYMENT = private-gateway-istio

help:
	@echo "$(BLUE)=== Istio Mesh Test Makefile ===$(NC)"
	@echo ""
	@echo "Usage:"
	@echo "  make <target> [NAMESPACE=ns] [ISTIO_NAMESPACE=istio-system]"
	@echo ""
	@echo "Targets:"
	@echo "  test                  - Run full Istio mesh test flow"
	@echo "    test-start-message   - Print start message"
	@echo "    validate-input       - Validate required parameters"
	@echo "    check-ztunnel-logs    - Validate ztunnel logs for NAMESPACE"
	@echo "    check-gateway-services - Verify public/private services point to istio gateway deployments"
	@echo "    run-mesh-test         - Run mesh smoke test (sub-make)"
	@echo "    check-public-gateway-istio-logs - Verify gateway received /mesh-test/health"
	@echo "    check-waypoint-logs   - Verify waypoint received /mesh-test/health"
	@echo "  collect-logs      - Collect last N lines from ztunnel, waypoint, public-gateway, public-frontend-gateway"
	@echo ""
	@echo "Variables:"
	@echo "  NAMESPACE        - Namespace under test (required)"
	@echo "  ISTIO_NAMESPACE  - Istio control-plane namespace (required)"
	@echo "  LOG_TAIL_LINES   - Number of log lines to collect per component (default: 100)"
	@echo "  LOG_OUTPUT_DIR   - Base directory for log files (default: ./istio-logs); logs go to LOG_OUTPUT_DIR/NAMESPACE/"
	@echo ""
	@echo "Runs mesh smoke test for the given namespace."
	@echo "The test verifies that traffic goes through Istio mesh: ztunnel, public-gateway, waypoint."
	@echo "Confirms that fallback route from Istio public gateway to core-mesh public gateway is working fine."
 
# Validate input parameters
validate-input:
	@echo "$(BLUE)=== Validating input parameters...$(NC)"
	@if [ -z "$(NAMESPACE)" ]; then \
		echo "$(RED)Error: NAMESPACE is empty. Set NAMESPACE for smoke test.$(NC)"; \
		exit 1; \
	fi;
	@if [ -z "$(ISTIO_NAMESPACE)" ]; then \
		echo "$(RED)Error: ISTIO_NAMESPACE is empty. Set ISTIO_NAMESPACE for smoke test.$(NC)"; \
		exit 1; \
	fi;
	@echo "$(CYAN)--- NAMESPACE: $(NAMESPACE)$(NC)"
	@echo "$(CYAN)--- ISTIO_NAMESPACE: $(ISTIO_NAMESPACE)$(NC)"
	@echo "$(GREEN)=== Input parameters validated successfully$(NC)"
	@echo ""

# Validate that the service endpoints belong to the expected deployment
check-gateway-services:
	@check_service_targets() { \
		namespace="$$1"; \
		service="$$2"; \
		deployment="$$3"; \
		echo "$(BLUE)=== Checking service $$service targets deployment $$deployment ===$(NC)"; \
		PODS=$$(kubectl get endpoints -n "$$namespace" "$$service" -o jsonpath='{.subsets[*].addresses[*].targetRef.name}'); \
		if [ -z "$$PODS" ]; then \
			echo "$(RED)Error: No endpoints found for service $$service in $$namespace$(NC)"; \
			exit 1; \
		fi; \
		for pod in $$PODS; do \
			rs=$$(kubectl get pod -n "$$namespace" $$pod -o jsonpath='{.metadata.ownerReferences[0].name}'); \
			dep=$$(kubectl get rs -n "$$namespace" $$rs -o jsonpath='{.metadata.ownerReferences[0].name}'); \
			if [ "$$dep" != "$$deployment" ]; then \
				echo "$(RED)Error: Service $$service points to deployment $$dep (expected $$deployment)$(NC)"; \
				exit 1; \
			fi; \
		done; \
		echo "$(GREEN)=== Service $$service points to deployment $$deployment ===$(NC)"; \
		echo ""; \
	}; \
	check_service_targets "$(NAMESPACE)" "$(PUBLIC_GATEWAY_SERVICE)" "$(PUBLIC_GATEWAY_DEPLOYMENT)"; \
	check_service_targets "$(NAMESPACE)" "$(PRIVATE_GATEWAY_SERVICE)" "$(PRIVATE_GATEWAY_DEPLOYMENT)";


define CHECK_ZTUNNEL_LOGS_FUNC
check_ztunnel_logs() { \
	namespace="$$1"; \
	echo "$(YELLOW)--- Checking ztunnel logs for namespace $$namespace...$(NC)"; \
	ZTUNNEL_POD=$$(kubectl get pod -n $(ISTIO_NAMESPACE) -l app=ztunnel -o jsonpath='{.items[0].metadata.name}'); \
	LOG_MATCH=$$(kubectl logs -n $(ISTIO_NAMESPACE) $$ZTUNNEL_POD | grep -i "$$namespace/sa/.*$$namespace/sa/.*"); \
	if [ -z "$$LOG_MATCH" ]; then \
		echo "$(RED)Error: ztunnel logs do not contain expected $$namespace/sa entries$(NC)"; \
		exit 1; \
	fi; \
	echo "$$LOG_MATCH" | tail -n 1; \
};
endef

check-ztunnel-logs:
	@$(CHECK_ZTUNNEL_LOGS_FUNC) \
	echo "$(BLUE)=== Checking ztunnel logs$(NC)"; \
	check_ztunnel_logs "$(NAMESPACE)"; \
	echo "$(GREEN)=== ztunnel logs are fine$(NC)"; \
	echo ""; \

run-mesh-test:
	@echo "$(CYAN)--- Running mesh test...$(NC)"
	@'$(MAKE)' -C ../../test/mesh-test mesh-smoke-test NAMESPACE=$(NAMESPACE)

check-public-gateway-istio-logs:
	@echo "$(CYAN)--- Checking public gateway istio logs...$(NC)"
	@PODS=$$(kubectl get pods -n $(NAMESPACE) -l gateway.networking.k8s.io/gateway-name=public-gateway -o jsonpath='{.items[*].metadata.name}'); \
	if [ -z "$$PODS" ]; then \
		echo "$(RED)Error: public-gateway-istio pod not found in namespace $(NAMESPACE)$(NC)"; \
		exit 1; \
	fi; \
	POD=$$(echo "$$PODS" | awk '{print $$1}'); \
	LOG_MATCH=$$(kubectl logs -n $(NAMESPACE) "$$POD" | grep -F "/mesh-test/health"); \
	if [ -z "$$LOG_MATCH" ]; then \
		echo "$(RED)Error: no /mesh-test/health entries in public gateway logs$(NC)"; \
		exit 1; \
	fi; \
	echo "$$LOG_MATCH" | tail -n 1; \
	echo "$(GREEN)=== Gateway interaction verified (/mesh-test/health found)$(NC)"
	@echo ""

check-waypoint-logs:
	@echo "$(CYAN)--- Checking waypoint logs...$(NC)"
	@PODS=$$(kubectl get pods -n $(NAMESPACE) -l gateway.networking.k8s.io/gateway-name=waypoint -o jsonpath='{.items[*].metadata.name}'); \
	if [ -z "$$PODS" ]; then \
		echo "$(RED)Error: waypoint pod not found in namespace $(NAMESPACE)$(NC)"; \
		exit 1; \
	fi; \
	POD=$$(echo "$$PODS" | awk '{print $$1}'); \
	LOG_MATCH=$$(kubectl logs -n $(NAMESPACE) "$$POD" | grep -F "/mesh-test/health"); \
	if [ -z "$$LOG_MATCH" ]; then \
		echo "$(RED)Error: no /mesh-test/health entries in waypoint logs$(NC)"; \
		exit 1; \
	fi; \
	echo "$$LOG_MATCH" | tail -n 1; \
	echo "$(GREEN)=== Gateway interaction verified (/mesh-test/health found)$(NC)"
	@echo ""

test-start-message:
	@echo "$(BLUE)=== Starting Istio Mesh test for namespace \"$(NAMESPACE)\"(NC)"
	@echo ""

collect-logs: validate-input
	@mkdir -p $(LOG_OUTPUT_PATH)
	@echo "$(BLUE)=== Collecting logs (last $(LOG_TAIL_LINES) lines) → $(LOG_OUTPUT_PATH)/ ===$(NC)"
	@echo ""
	@echo "--- ztunnel ($(ISTIO_NAMESPACE)) ---" > $(LOG_OUTPUT_PATH)/ztunnel.log
	@for pod in $$(kubectl get pods -n $(ISTIO_NAMESPACE) -l app=ztunnel -o jsonpath='{.items[*].metadata.name}'); do \
		[ -n "$$pod" ] || continue; \
		echo ">>> $$pod" >> $(LOG_OUTPUT_PATH)/ztunnel.log; \
		kubectl logs -n $(ISTIO_NAMESPACE) "$$pod" --tail=$(LOG_TAIL_LINES) 2>/dev/null >> $(LOG_OUTPUT_PATH)/ztunnel.log || echo "(no logs)" >> $(LOG_OUTPUT_PATH)/ztunnel.log; \
		echo "" >> $(LOG_OUTPUT_PATH)/ztunnel.log; \
	done
	@echo "--- waypoint ($(NAMESPACE)) ---" > $(LOG_OUTPUT_PATH)/waypoint.log
	@for pod in $$(kubectl get pods -n $(NAMESPACE) -l gateway.networking.k8s.io/gateway-name=waypoint -o jsonpath='{.items[*].metadata.name}'); do \
		[ -n "$$pod" ] || continue; \
		echo ">>> $$pod" >> $(LOG_OUTPUT_PATH)/waypoint.log; \
		kubectl logs -n $(NAMESPACE) "$$pod" --tail=$(LOG_TAIL_LINES) 2>/dev/null >> $(LOG_OUTPUT_PATH)/waypoint.log || echo "(no logs)" >> $(LOG_OUTPUT_PATH)/waypoint.log; \
		echo "" >> $(LOG_OUTPUT_PATH)/waypoint.log; \
	done
	@echo "--- public-gateway ($(NAMESPACE)) ---" > $(LOG_OUTPUT_PATH)/public-gateway.log
	@for pod in $$(kubectl get pods -n $(NAMESPACE) -l gateway.networking.k8s.io/gateway-name=public-gateway -o jsonpath='{.items[*].metadata.name}'); do \
		[ -n "$$pod" ] || continue; \
		echo ">>> $$pod" >> $(LOG_OUTPUT_PATH)/public-gateway.log; \
		kubectl logs -n $(NAMESPACE) "$$pod" --tail=$(LOG_TAIL_LINES) 2>/dev/null >> $(LOG_OUTPUT_PATH)/public-gateway.log || echo "(no logs)" >> $(LOG_OUTPUT_PATH)/public-gateway.log; \
		echo "" >> $(LOG_OUTPUT_PATH)/public-gateway.log; \
	done
	@echo "--- public-frontend-gateway ($(NAMESPACE)) ---" > $(LOG_OUTPUT_PATH)/public-frontend-gateway.log
	@for pod in $$(kubectl get pods -n $(NAMESPACE) -l name=public-frontend-gateway -o jsonpath='{.items[*].metadata.name}'); do \
		[ -n "$$pod" ] || continue; \
		echo ">>> $$pod" >> $(LOG_OUTPUT_PATH)/public-frontend-gateway.log; \
		kubectl logs -n $(NAMESPACE) "$$pod" --tail=$(LOG_TAIL_LINES) 2>/dev/null >> $(LOG_OUTPUT_PATH)/public-frontend-gateway.log || echo "(no logs)" >> $(LOG_OUTPUT_PATH)/public-frontend-gateway.log; \
		echo "" >> $(LOG_OUTPUT_PATH)/public-frontend-gateway.log; \
	done
	@echo "$(GREEN)=== Logs written to:$(NC)"
	@echo "  $(LOG_OUTPUT_PATH)/ztunnel.log"
	@echo "  $(LOG_OUTPUT_PATH)/waypoint.log"
	@echo "  $(LOG_OUTPUT_PATH)/public-gateway.log"
	@echo "  $(LOG_OUTPUT_PATH)/public-frontend-gateway.log"

test: test-start-message validate-input check-ztunnel-logs check-gateway-services run-mesh-test check-public-gateway-istio-logs check-waypoint-logs
	@echo "$(GREEN)   - Ztunnel is registering traffic$(NC)"
	@echo "$(GREEN)   - Public-gateway-service is pointing to public-gateway-istio$(NC)"
	@echo "$(GREEN)   - Public-gateway-istio has received traffic for mesh test service$(NC)"
	@echo "$(GREEN)   - Fallback route on public-gateway-istio is working fine$(NC)"
	@echo "$(GREEN)   - Waypoint has received traffic for mesh test service$(NC)"
	@echo ""
	@echo "$(GREEN)=== Istio Mesh test completed for namespace \"$(NAMESPACE)\"(NC)"
