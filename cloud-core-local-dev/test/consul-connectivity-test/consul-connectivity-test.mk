# =============================================================================
# CONSUL TARGETS
# =============================================================================

# Phony targets
.PHONY: check-consul-connectivity

# Script file variable
CONSUL_CHECK_SCRIPT_FILE ?= test/consul-connectivity-test/consul-connectivity-test.sh
CONSUL_CHECK_SCRIPT := $(shell cat $(CONSUL_CHECK_SCRIPT_FILE))

# Check Consul connectivity
check-consul-connectivity:
	@if [ "$(CONSUL_ENABLED)" = "true" ]; then \
		echo "$(YELLOW)--- Checking Consul connectivity...$(NC)"; \
		echo "Checking Consul service availability..."; \
		if kubectl get svc -n $(CONSUL_NAMESPACE) $(CONSUL_SERVICE_NAME) >/dev/null 2>&1; then \
			echo "$(GREEN)? Consul service found$(NC)"; \
		else \
			echo "$(RED)? Consul service not found$(NC)"; \
			exit 1; \
		fi; \
		echo "Checking Consul pod status..."; \
		CONSUL_PODS=$$(kubectl get pods -n $(CONSUL_NAMESPACE) --selector=app=consul -o jsonpath='{.items[*].status.phase}'); \
		if echo "$$CONSUL_PODS" | grep -q "Running"; then \
			echo "$(GREEN)? Consul pods are running$(NC)"; \
		else \
			echo "$(RED)? Consul pods are not running: $$CONSUL_PODS$(NC)"; \
			exit 1; \
		fi; \
		echo "Creating temporary pod for Consul connectivity check..."; \
		echo "$(YELLOW)--- Retrying Consul connectivity check for up to 5 minutes...$(NC)"; \
		kubectl run consul-check-$$(date +%s) --image=curlimages/curl:latest -n $(CORE_NAMESPACE) --rm -i --restart=Never -- \
			sh -c "CONSUL_SERVICE_NAME=$(CONSUL_SERVICE_NAME); CONSUL_NAMESPACE=$(CONSUL_NAMESPACE); INGRESS_GATEWAY_CLOUD_PRIVATE_HOST=$(INGRESS_GATEWAY_CLOUD_PRIVATE_HOST); $(CONSUL_CHECK_SCRIPT)" || (echo "$(RED)✗ Consul connectivity check failed$(NC)" && exit 1); \
		echo "$(GREEN)=== Consul connectivity check completed successfully$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Consul connectivity check (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi 