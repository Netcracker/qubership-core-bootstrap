# Profile definition
PROFILE ?= cloud-core-dev

# Color definitions
YELLOW := \033[1;33m
GREEN := \033[1;32m
RED := \033[0;31m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m

minikube-setup: minikube-setup-print-start create-minikube-profile
	@echo "$(GREEN)=== Minikube setup completed...$(NC)"
	@echo ""

minikube-setup-print-start:
	@echo "$(GREEN)=== Starting Minikube setup...$(NC)"
	@echo ""

create-minikube-profile:
	@echo "$(YELLOW)--- Checking minikube profile status...$(NC)"
	@if minikube profile list | grep "$(PROFILE)" | grep -q "OK"; then \
		echo "Profile $(PROFILE) is already running"; \
	else \
		echo "Creating new profile $(PROFILE)..."; \
		minikube start --nodes=3 -p=$(PROFILE) --driver=docker; \
	fi
	@echo ""

minikube-delete-profile:
	@echo "$(YELLOW)--- Cleaning up resources...$(NC)"
	minikube delete -p=$(PROFILE)
	@echo ""
