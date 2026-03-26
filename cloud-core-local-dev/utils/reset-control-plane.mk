.PHONY: reset-control-plane delete-control-plane-database restart-services restart-public-gateway \
	restart-services-start restart-control-plane check-public-gateway-config-no-mesh-test


# =============================================================================
# Reset Control Plane
# =============================================================================
reset-control-plane: delete-control-plane-database deploy-cloud-core-configuration restart-services
	@echo "$(GREEN)=== CONTROL PLANE RESET SUCCESSFULLY ===$(NC)"

delete-control-plane-database:
	@echo "PG_NAMESPACE: $(PG_NAMESPACE)"
	@echo "CORE_NAMESPACE: $(CORE_NAMESPACE)"
	@echo "$(CYAN)--- Cleaning up: Removing control-plane database via REST (from cluster, no port-forward)...$(NC)"; \
	DBAAS_USERNAME=$$(kubectl -n $(PG_NAMESPACE) get secret dbaas-aggregator-registration-credentials -o jsonpath='{.data.username}' 2>/dev/null | base64 -d); \
	DBAAS_PASSWORD=$$(kubectl -n $(PG_NAMESPACE) get secret dbaas-aggregator-registration-credentials -o jsonpath='{.data.password}' 2>/dev/null | base64 -d); \
	BASIC_AUTH_HEADER=$$(printf "%s" "$$DBAAS_USERNAME:$$DBAAS_PASSWORD" | base64);\
	if [ -z "$$DBAAS_USERNAME" ] || [ -z "$$DBAAS_PASSWORD" ]; then \
		echo "$(YELLOW)--- DBaaS credentials not found, skipping database removal$(NC)"; \
	else \
		echo "$(BASIC_AUTH_HEADER)"; \
		kubectl exec -n $(CORE_NAMESPACE) deploy/control-plane -- \
		curl -s -w "HTTPSTATUS:%{http_code}" -X DELETE \
		"http://dbaas-aggregator.dbaas.svc.cluster.local:8080/api/v3/dbaas/$(CORE_NAMESPACE)/databases/postgresql" \
		-H "Authorization: Basic $$BASIC_AUTH_HEADER" \
		-H "Content-Type: application/json" \
		-d '{"classifier":{"microserviceName":"control-plane","scope":"service","namespace":"$(CORE_NAMESPACE)"},"originService":"control-plane"}'; \
	fi; \
	echo ""

restart-services: restart-services-start restart-control-plane restart-public-gateway check-public-gateway-config-no-mesh-test
	kubectl rollout restart deploy/paas-mediation -n $(CORE_NAMESPACE)
	kubectl rollout restart deploy/site-management -n $(CORE_NAMESPACE)
	kubectl rollout restart deploy/config-server -n $(CORE_NAMESPACE)
	kubectl rollout restart deploy/dbaas-agent -n $(CORE_NAMESPACE)
	kubectl rollout restart deploy/maas-agent -n $(CORE_NAMESPACE) 2>/dev/null || true
	kubectl rollout restart deploy/private-frontend-gateway -n $(CORE_NAMESPACE)
	kubectl rollout restart deploy/internal-gateway -n $(CORE_NAMESPACE)
	kubectl rollout restart deploy/core-operator -n $(CORE_NAMESPACE)
	kubectl rollout restart deploy/facade-operator -n $(CORE_NAMESPACE)
	@echo "$(GREEN)=== ALL SERVICES RESTARTED SUCCESSFULLY ===$(NC)"

restart-services-start:
	@echo "$(CYAN)--- Restarting all services...$(NC)"

restart-control-plane:
	@$(call rollout_restart_and_wait,control-plane)

restart-public-gateway:
	@$(call rollout_restart_and_wait,public-frontend-gateway)

check-public-gateway-config-no-mesh-test:
	@NEW_RS=$$(kubectl get rs -n $(CORE_NAMESPACE) -l name=public-frontend-gateway \
		--sort-by=.metadata.creationTimestamp \
		-o jsonpath='{.items[-1].metadata.name}' 2>/dev/null); \
	if [ -z "$$NEW_RS" ]; then \
		echo "$(RED)No ReplicaSet found for public-frontend-gateway in $(CORE_NAMESPACE)$(NC)"; exit 1; \
	fi; \
	echo "$(CYAN)--- Using ReplicaSet: $$NEW_RS$(NC)"; \
	POD=$$(kubectl get pods -n $(CORE_NAMESPACE) \
		-o jsonpath='{.items[?(@.status.containerStatuses[0].ready==true)].metadata.name}' 2>/dev/null \
		| tr ' ' '\n' \
		| while read p; do \
			RS=$$(kubectl get pod -n $(CORE_NAMESPACE) "$$p" \
				-o jsonpath='{.metadata.ownerReferences[?(@.kind=="ReplicaSet")].name}' 2>/dev/null); \
			[ "$$RS" = "$$NEW_RS" ] && echo "$$p" && break; \
		done); \
	if [ -z "$$POD" ]; then \
		echo "$(RED)No ready pod found in new RS $$NEW_RS$(NC)"; exit 1; \
	fi; \
	POD_IP=$$(kubectl get pod -n $(CORE_NAMESPACE) "$$POD" -o jsonpath='{.status.podIP}'); \
	if [ -z "$$POD_IP" ]; then \
		echo "$(RED)Could not determine pod IP for $$POD$(NC)"; exit 1; \
	fi; \
	echo "$(CYAN)--- Using pod $$POD ($$POD_IP) from new RS $$NEW_RS$(NC)"; \
	echo "$(CYAN)--- Checking public-frontend-gateway Envoy config_dump for 'mesh-test'...$(NC)"; \
	CONFIG=$$(kubectl exec -n $(CORE_NAMESPACE) deploy/control-plane -- curl -s "$$POD_IP:9901/config_dump"); \
	echo "$$CONFIG" | grep "mesh-test"; \
	if echo "$$CONFIG" | grep -q "mesh-test"; then \
		echo "$(RED) ✗ 'mesh-test' found in public-frontend-gateway config$(NC)"; exit 1; \
	fi; \
	echo "$(GREEN) ✓ 'mesh-test' not found in public-frontend-gateway config_dump$(NC)";

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

# $(1) - deployment name
define rollout_restart_and_wait
	echo "$(CYAN)--- Restarting $(1)...$(NC)"; \
	OLD_RS=$$(kubectl get rs -n $(CORE_NAMESPACE) \
		-l name=$(1) \
		--sort-by=.metadata.creationTimestamp \
		-o jsonpath='{.items[-1].metadata.name}' 2>/dev/null); \
	echo "$(CYAN)--- Old ReplicaSet: $${OLD_RS:-none}$(NC)"; \
	kubectl rollout restart deploy/$(1) -n $(CORE_NAMESPACE); \
	echo "$(CYAN)--- Waiting for old RS $$OLD_RS to scale to 0...$(NC)"; \
	TIMEOUT=300; \
	START=$$(date +%s); \
	while [ -n "$$OLD_RS" ]; do \
		ELAPSED=$$(( $$(date +%s) - START )); \
		if [ "$$ELAPSED" -ge "$$TIMEOUT" ]; then \
			echo "$(RED)Timeout: old RS $$OLD_RS still has replicas after $${TIMEOUT}s$(NC)"; exit 1; \
		fi; \
		REPLICAS=$$(kubectl get rs -n $(CORE_NAMESPACE) "$$OLD_RS" \
			-o jsonpath='{.status.replicas}' 2>/dev/null); \
		if [ -z "$$REPLICAS" ] || [ "$$REPLICAS" = "0" ]; then \
			echo "$(CYAN)--- Old RS $$OLD_RS scaled to 0 (after $${ELAPSED}s)$(NC)"; \
			break; \
		fi; \
		echo "  [$${ELAPSED}s] Old RS $$OLD_RS still has $$REPLICAS replica(s)... ($$(( TIMEOUT - ELAPSED ))s remaining)"; \
		sleep 3; \
	done; \
	echo "$(CYAN)--- Waiting for new RS to be fully ready...$(NC)"; \
	kubectl rollout status deploy/$(1) -n $(CORE_NAMESPACE) --timeout=5m; \
	echo "$(GREEN)--- Rollout of $(1) complete — all old pods gone, new pods ready$(NC)";
endef
