# tests/load.mk
# Load testing (k6 + bash). Эти цели вызывают существующие скрипты в tests/load/.

.PHONY: test-load-demo

test-load-demo:
	@echo "$(BLUE)Running k6 demo load tests...$(NC)"
	@bash tests/load/k6_tests/run-demo-tests.sh
	@echo "$(GREEN)✓ Load demo tests completed.$(NC)"
