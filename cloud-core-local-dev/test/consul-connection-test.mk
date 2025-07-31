# =============================================================================
# CONSUL TARGETS
# =============================================================================

# Phony targets
.PHONY: check-consul-connectivity

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
			sh -c "echo 'Starting Consul connectivity check with retry loop...' && \
			TIMEOUT=300 && \
			START_TIME=\$$(date +%s) && \
			while true; do \
				CURRENT_TIME=\$$(date +%s) && \
				ELAPSED_TIME=\$$((CURRENT_TIME - START_TIME)) && \
				if [ \$$ELAPSED_TIME -ge \$$TIMEOUT ]; then \
					echo 'Timeout reached after 5 minutes. Consul connectivity check failed.' && \
					exit 1; \
				fi && \
				echo \"Attempt \$$((ELAPSED_TIME / 10 + 1)) - Checking Consul connectivity... (\$$((TIMEOUT - ELAPSED_TIME))s remaining)\" && \
				if curl -s -f http://$(CONSUL_SERVICE_NAME).$(CONSUL_NAMESPACE).${INGRESS_GATEWAY_CLOUD_PRIVATE_HOST}:8500/v1/status/leader >/dev/null 2>&1; then \
					echo '? Consul API is responding' && \
					if curl -s -f http://$(CONSUL_SERVICE_NAME).$(CONSUL_NAMESPACE).${INGRESS_GATEWAY_CLOUD_PRIVATE_HOST}:8500/v1/status/peers >/dev/null 2>&1; then \
						echo '? Consul cluster is healthy' && \
						echo '? Consul connectivity check successful!' && \
						exit 0; \
					else \
						echo '? Consul cluster health check failed' && \
						echo 'Retrying in 10 seconds...' && \
						sleep 10; \
					fi; \
				else \
					echo '? Consul API is not responding' && \
					echo 'Retrying in 10 seconds...' && \
					sleep 10; \
				fi; \
			done" || (echo "$(RED)? Consul connectivity check failed$(NC)" && exit 1); \
		echo "$(GREEN)=== Consul connectivity check completed successfully$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Consul connectivity check (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi 