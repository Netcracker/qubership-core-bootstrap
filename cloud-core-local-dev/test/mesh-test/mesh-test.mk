# =============================================================================
# MESH TESTING TARGETS
# =============================================================================

# Phony targets
.PHONY: mesh-smoke-test mesh-smoke-test-print-start deploy-mesh-test wait-for-mesh-test-pod run-mesh-test-curl cleanup-mesh-test

MESH_TEST_YAML_FILE ?= test/mesh-test/mesh-test.yaml

# Main mesh smoke test target
mesh-smoke-test: mesh-smoke-test-print-start deploy-mesh-test wait-for-mesh-test-pod run-mesh-test-curl cleanup-mesh-test
	@echo "$(GREEN)=== Cloud Core Mesh Smoke Test completed$(NC)"
	@echo ""

# Print start message for mesh smoke test
mesh-smoke-test-print-start:
	@echo "$(GREEN)=== Starting Cloud Core Mesh Smoke Test...$(NC)"
	@echo ""

# Deploy mesh test resources
deploy-mesh-test:
	@echo "$(YELLOW)--- Deploying mesh test resources...$(NC)"
	@if [ -f "$(MESH_TEST_YAML_FILE)" ]; then \
		kubectl apply -f $(MESH_TEST_YAML_FILE) -n $(CORE_NAMESPACE); \
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
			kubectl get pods -n $(CORE_NAMESPACE) --selector=app=mesh-test -o wide; \
			echo "$(CYAN)Pod logs:$(NC)"; \
			kubectl logs -n $(CORE_NAMESPACE) --selector=app=mesh-test --tail=50; \
			exit 1; \
		fi; \
		READY_PODS=$$(kubectl get pods -n $(CORE_NAMESPACE) --selector=app=mesh-test --field-selector=status.phase=Running -o jsonpath='{.items[?(@.status.containerStatuses[0].ready==true)].metadata.name}' | wc -w); \
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
	@kubectl run mesh-test-curl-$$(date +%s) --image=curlimages/curl:latest -n $(CORE_NAMESPACE) --rm -i --restart=Never -- \
		sh -c "echo 'Testing mesh service through public gateway...' && \
		echo 'Attempting to call mesh test service...' && \
		TIMEOUT=60 && \
		START_TIME=\$$(date +%s) && \
		while true; do \
			CURRENT_TIME=\$$(date +%s) && \
			ELAPSED_TIME=\$$((CURRENT_TIME - START_TIME)) && \
			if [ \$$ELAPSED_TIME -ge \$$TIMEOUT ]; then \
				echo 'Timeout reached after 60 seconds. Mesh test failed.' && \
				exit 1; \
			fi && \
			echo \"Attempt \$$((ELAPSED_TIME / 5 + 1)) - Testing mesh service... (\$$((TIMEOUT - ELAPSED_TIME))s remaining)\" && \
			if curl -s -f -m 10 http://mesh-test-service:8080/health >/dev/null 2>&1; then \
				echo '? Mesh service is responding internally' && \
				curl -s -m 10 http://public-gateway-service:8080/mesh-test/health && \
				if curl -f -m 10 http://public-gateway-service:8080/mesh-test/health >/dev/null 2>&1; then \
					echo '? Mesh service is accessible through public gateway' && \
					echo '? Mesh smoke test successful!' && \
					echo 'Response from public gateway:' && \
					curl -s -m 10 http://public-gateway-service:8080/mesh-test/health && \
					echo '' && \
					exit 0; \
				else \
					echo '? Mesh service not accessible through public gateway' && \
					echo 'Retrying in 5 seconds...' && \
					sleep 5; \
				fi; \
			else \
				echo '? Mesh service not responding internally' && \
				echo 'Retrying in 5 seconds...' && \
				sleep 5; \
			fi; \
		done" || (echo "$(RED)? Mesh smoke test failed$(NC)" && exit 1)
	@echo "$(GREEN)=== Mesh smoke test completed successfully$(NC)"
	@echo ""

# Clean up mesh test resources
cleanup-mesh-test:
	@echo "$(YELLOW)--- Cleaning up mesh test resources...$(NC)"
	@if [ -f "$(MESH_TEST_YAML_FILE)" ]; then \
		echo "$(CYAN)--- Deleting mesh test resources from $(CORE_NAMESPACE) namespace...$(NC)"; \
		kubectl delete -f $(MESH_TEST_YAML_FILE) -n $(CORE_NAMESPACE) --ignore-not-found=true || true; \
		echo "$(GREEN)--- Mesh test resources cleaned up successfully$(NC)"; \
	fi
	@echo "" 