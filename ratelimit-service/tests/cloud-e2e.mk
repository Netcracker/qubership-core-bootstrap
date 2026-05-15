# tests/cloud-e2e.mk
# Cloud end-to-end (requires a deployed service in the cluster).

.PHONY: test-cloud-e2e test-cloud-e2e-performance test-cloud-e2e-layered

test-cloud-e2e:
	@echo "$(BLUE)Running cloud e2e tests...$(NC)"
	@$(PF_SCRIPT) --profile=cloud --start
	@go clean -testcache
	@TEST_JWT_TOKEN=$(TEST_JWT_TOKEN) \
		go test -v -tags=cloud_e2e -timeout $(TEST_TIMEOUT) ./tests/cloud-e2e/... ; \
		EXIT=$$? ; \
		$(PF_SCRIPT) --profile=cloud --stop ; \
		exit $$EXIT
	@echo "$(GREEN)✓ Cloud e2e tests passed!$(NC)"

test-cloud-e2e-performance:
	@echo "$(BLUE)Running cloud e2e performance subset...$(NC)"
	@$(PF_SCRIPT) --profile=cloud --start
	@go clean -testcache
	@TEST_JWT_TOKEN=$(TEST_JWT_TOKEN) \
		go test -v -tags=cloud_e2e -run TestCloudE2E_Performance -timeout $(TEST_TIMEOUT) ./tests/cloud-e2e/... ; \
		EXIT=$$? ; \
		$(PF_SCRIPT) --profile=cloud --stop ; \
		exit $$EXIT
	@echo "$(GREEN)✓ Cloud e2e performance subset passed!$(NC)"

test-cloud-e2e-layered:
	@echo "$(BLUE)Running cloud e2e composite-limits test (long-running ~2 min)...$(NC)"
	@$(PF_SCRIPT) --profile=cloud --start
	@go clean -testcache
	@TEST_JWT_TOKEN=$(TEST_JWT_TOKEN) \
		go test -v -tags=cloud_e2e_layered -timeout 5m ./tests/cloud-e2e/... ; \
		EXIT=$$? ; \
		$(PF_SCRIPT) --profile=cloud --stop ; \
		exit $$EXIT
	@echo "$(GREEN)✓ Cloud e2e composite-limits test passed!$(NC)"
