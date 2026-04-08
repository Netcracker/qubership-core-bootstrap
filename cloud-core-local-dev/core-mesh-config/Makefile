# Core Mesh Config Makefile
# Handles enabling/disabling of Istio ambient mesh for application namespaces
# via qubership-core-mesh-config Helm chart.

# Include configuration
CONFIG_FILE ?= local.mk
include $(CONFIG_FILE)

# Repository directory
REPOS_DIR ?= ./.tmp

# Define colors for output
YELLOW := \033[1;33m
GREEN := \033[1;32m
RED := \033[0;31m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m

.PHONY: help \
	setup-repo \
	enable disable \
	label-namespaces unlabel-namespaces

help:
	@echo "$(BLUE)=== Core Mesh Config Makefile ===$(NC)"
	@echo ""
	@echo "Usage: make <target> MESH_NAMESPACES=ns1,ns2 [CONFIG_FILE=local.mk]"
	@echo ""
	@echo "Targets:"
	@echo "  enable                   - Clone repo, label namespaces, install Helm chart, run smoke test"
	@echo "    setup-repo             - Clone/update qubership-core-mesh-config repository"
	@echo "    label-namespaces       - Apply Istio ambient/waypoint labels to MESH_NAMESPACES (no Helm)"
	@echo "  disable                  - Uninstall Helm chart, remove labels, restore gateway selectors"
	@echo "    unlabel-namespaces     - Remove Istio labels and restore gateway selectors (no Helm)"
	@echo ""
	@echo "Variables:"
	@echo "  MESH_NAMESPACES          - Comma-separated namespaces to enable mesh on (required)"
	@echo "  ISTIO_NAMESPACE          - Istio control-plane namespace (default: istio-system)"
	@echo "  CONFIG_FILE              - Configuration file to use (default: local.mk)"
	@echo "  CORE_MESH_CONFIG_REPO_URL    - Git URL for qubership-core-mesh-config"
	@echo "  CORE_MESH_CONFIG_REPO_BRANCH - Branch to use (default: main)"
	@echo "  MESH_HELM_RELEASE_NAME   - Helm release name (default: core-istio-mesh)"
	@echo "  MESH_HELM_EXTRA_ARGS     - Extra helm args for mesh chart"
	@echo "  RUN_SMOKE_TEST           - Run smoke test after enable (default: true)"

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

# $(1) - namespace
define label_namespace_for_mesh
	echo "$(CYAN)--- Labeling namespace $(1) for mesh$(NC)"; \
	kubectl label namespace "$(1)" istio.io/dataplane-mode=ambient --overwrite; \
	kubectl label namespace "$(1)" istio.io/use-waypoint=waypoint --overwrite;
endef

# $(1) - namespace
define unlabel_namespace_for_mesh
	echo "$(CYAN)--- Removing mesh labels from namespace $(1)$(NC)"; \
	kubectl label namespace "$(1)" istio.io/dataplane-mode- --overwrite; \
	kubectl label namespace "$(1)" istio.io/use-waypoint- --overwrite; \
	echo "$(CYAN)--- Restoring gateway service selectors in namespace $(1)$(NC)"; \
	kubectl patch service public-gateway-service -n "$(1)" --type='json' -p='[{"op": "replace", "path": "/spec/selector", "value": {"name":"public-frontend-gateway"}}]' || true; \
	kubectl patch service private-gateway-service -n "$(1)" --type='json' -p='[{"op": "replace", "path": "/spec/selector", "value": {"name":"private-frontend-gateway"}}]' || true;
endef

# $(1) - namespace
define install_mesh_helm
	echo "$(CYAN)--- Installing $(MESH_HELM_RELEASE_NAME) Helm package in $(1)$(NC)"; \
	echo "MESH_HELM_EXTRA_ARGS: $(MESH_HELM_EXTRA_ARGS)"; \
	helm upgrade --install $(MESH_HELM_RELEASE_NAME) $(REPOS_DIR)/qubership-core-mesh-config/ \
		-f $(REPOS_DIR)/qubership-core-mesh-config/resource-profiles/dev.yaml \
		--namespace "$(1)" \
		--set deploy_feature_flags.core.serviceMeshType=ISTIO \
		--wait \
		$(MESH_HELM_EXTRA_ARGS);
endef

# $(1) - namespace
define uninstall_mesh_helm
	echo "$(CYAN)--- Uninstalling $(MESH_HELM_RELEASE_NAME) Helm package in $(1)$(NC)"; \
	helm uninstall $(MESH_HELM_RELEASE_NAME) -n "$(1)" --ignore-not-found=true || true;
endef

# =============================================================================
# TARGETS
# =============================================================================

setup-repo:
	@echo "$(YELLOW)--- Setting up qubership-core-mesh-config repository...$(NC)"
	@mkdir -p $(REPOS_DIR)
	@if [ -d "$(REPOS_DIR)/qubership-core-mesh-config" ]; then \
		CURRENT_BRANCH=$$(cd $(REPOS_DIR)/qubership-core-mesh-config && git rev-parse --abbrev-ref HEAD); \
		if [ "$$CURRENT_BRANCH" != "$(CORE_MESH_CONFIG_REPO_BRANCH)" ]; then \
			echo "$(CYAN)--- Switching qubership-core-mesh-config from $$CURRENT_BRANCH to $(CORE_MESH_CONFIG_REPO_BRANCH)...$(NC)"; \
			(cd $(REPOS_DIR)/qubership-core-mesh-config && git fetch origin && git checkout $(CORE_MESH_CONFIG_REPO_BRANCH) && git pull); \
		else \
			echo "$(CYAN)--- Updating qubership-core-mesh-config (branch: $(CORE_MESH_CONFIG_REPO_BRANCH))...$(NC)"; \
			(cd $(REPOS_DIR)/qubership-core-mesh-config && git pull); \
		fi; \
	else \
		echo "$(CYAN)--- Cloning qubership-core-mesh-config (branch: $(CORE_MESH_CONFIG_REPO_BRANCH))...$(NC)"; \
		git clone -b $(CORE_MESH_CONFIG_REPO_BRANCH) $(CORE_MESH_CONFIG_REPO_URL) $(REPOS_DIR)/qubership-core-mesh-config; \
	fi
	@echo ""

label-namespaces:
	@if [ -z "$(MESH_NAMESPACES)" ]; then \
		echo "$(YELLOW)--- Skipping labeling (MESH_NAMESPACES is empty)...$(NC)"; \
	else \
		echo "$(BLUE)=== Labeling namespaces for mesh: $(MESH_NAMESPACES)$(NC)"; \
		for ns in $$(echo "$(MESH_NAMESPACES)" | tr ',' ' '); do \
			if [ -n "$$ns" ]; then \
				$(call label_namespace_for_mesh,$$ns) \
				echo "$(GREEN)--- Namespace $$ns labeled$(NC)"; \
				echo ""; \
			fi; \
		done; \
		echo "$(GREEN)=== All namespaces labeled$(NC)"; \
	fi

unlabel-namespaces:
	@if [ -z "$(MESH_NAMESPACES)" ]; then \
		echo "$(YELLOW)--- Skipping unlabeling (MESH_NAMESPACES is empty)...$(NC)"; \
	else \
		echo "$(BLUE)=== Unlabeling namespaces: $(MESH_NAMESPACES)$(NC)"; \
		for ns in $$(echo "$(MESH_NAMESPACES)" | tr ',' ' '); do \
			if [ -n "$$ns" ]; then \
				$(call unlabel_namespace_for_mesh,$$ns) \
				echo "$(GREEN)--- Namespace $$ns unlabeled$(NC)"; \
				echo ""; \
			fi; \
		done; \
		echo "$(GREEN)=== All namespaces unlabeled$(NC)"; \
	fi

enable: setup-repo
	@if [ -z "$(MESH_NAMESPACES)" ]; then \
		echo "$(YELLOW)--- Skipping mesh enablement (MESH_NAMESPACES is empty)...$(NC)"; \
	else \
		echo "$(BLUE)=== Enabling mesh for namespaces: $(MESH_NAMESPACES)$(NC)"; \
		for ns in $$(echo "$(MESH_NAMESPACES)" | tr ',' ' '); do \
			if [ -n "$$ns" ]; then \
				$(call label_namespace_for_mesh,$$ns) \
				$(call install_mesh_helm,$$ns) \
				if [ "$(RUN_SMOKE_TEST)" = "true" ]; then \
					'$(MAKE)' -C ../istio/test full-test NAMESPACE="$$ns" ISTIO_NAMESPACE="$(ISTIO_NAMESPACE)" || { '$(MAKE)' -C ../istio/test collect-logs NAMESPACE="$$ns" ISTIO_NAMESPACE="$(ISTIO_NAMESPACE)"; exit 1; }; \
				else \
					echo "$(YELLOW)--- Skipping smoke test (RUN_SMOKE_TEST is not true)$(NC)"; \
				fi; \
				echo ""; \
			fi; \
		done; \
		echo "$(GREEN)=== Mesh enabled for all namespaces$(NC)"; \
	fi

disable:
	@if [ -z "$(MESH_NAMESPACES)" ]; then \
		echo "$(YELLOW)--- Skipping mesh disabling (MESH_NAMESPACES is empty)...$(NC)"; \
	else \
		echo "$(BLUE)=== Disabling mesh for namespaces: $(MESH_NAMESPACES)$(NC)"; \
		for ns in $$(echo "$(MESH_NAMESPACES)" | tr ',' ' '); do \
			if [ -n "$$ns" ]; then \
				$(call uninstall_mesh_helm,$$ns) \
				$(call unlabel_namespace_for_mesh,$$ns) \
				echo "$(GREEN)--- Mesh disabled for namespace $$ns$(NC)"; \
				echo ""; \
			fi; \
		done; \
		echo "$(GREEN)=== Mesh disabled for all namespaces$(NC)"; \
	fi
