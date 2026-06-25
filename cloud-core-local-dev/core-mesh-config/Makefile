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
		--set SERVICE_MESH_TYPE=Istio \
		--set WAYPOINT_CPU_REQUEST=100m \
		--set CPU_REQUEST=10m \
		--set MONITORING_ENABLED=$(MONITORING_ENABLED) \
		--wait \
		--timeout 10m \
		--debug \
		$(MESH_HELM_EXTRA_ARGS);
endef

# $(1) - namespace
define dump_mesh_diagnostics
	echo "$(RED)--- Mesh install failed in $(1); dumping diagnostics$(NC)"; \
	echo "$(CYAN)--- Pods in $(1)$(NC)"; \
	kubectl get pods -n "$(1)" -o wide || true; \
	echo "$(CYAN)--- Pods in $(ISTIO_NAMESPACE)$(NC)"; \
	kubectl get pods -n "$(ISTIO_NAMESPACE)" -o wide || true; \
	echo "$(CYAN)--- All pods$(NC)"; \
	kubectl get pods -A -o wide || true; \
	echo "$(CYAN)--- Describe pods in $(1)$(NC)"; \
	kubectl describe pods -n "$(1)" || true; \
	echo "$(CYAN)--- Events in $(1)$(NC)"; \
	kubectl get events -n "$(1)" --sort-by=.lastTimestamp || true; \
	echo "$(CYAN)--- Gateways in $(1) (did one reappear during the wait?)$(NC)"; \
	kubectl get gateway.gateway.networking.k8s.io -n "$(1)" \
		-o custom-columns=NAME:.metadata.name,CLASS:.spec.gatewayClassName,DELETING:.metadata.deletionTimestamp,FINALIZERS:.metadata.finalizers 2>/dev/null || true; \
	kubectl get gateway.gateway.networking.k8s.io egress-gateway -n "$(1)" -o yaml 2>/dev/null || true; \
	echo "$(CYAN)--- HTTPRoutes / Services in $(1)$(NC)"; \
	kubectl get httproute.gateway.networking.k8s.io,svc -n "$(1)" 2>/dev/null || true; \
	echo "$(CYAN)--- Helm status$(NC)"; \
	helm status $(MESH_HELM_RELEASE_NAME) -n "$(1)" || true;
endef

# $(1) - namespace
# Pre-delete ALL leftover Gateways from prior runs before mesh install. Helm's
# before-hook-creation deletes existing hook Gateways and waits for them to vanish;
# if istiod can't clear the finalizer the wait blocks 10m. Stripping finalizers
# first guarantees deletion completes immediately, so helm's hook-delete finds none.
define clear_stuck_gateways
	echo "$(CYAN)--- Pre-cleaning leftover Gateways in $(1)$(NC)"; \
	for gw in $$(kubectl get gateway.gateway.networking.k8s.io -n "$(1)" \
		-o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null); do \
		echo "$(YELLOW)--- Removing finalizers and deleting Gateway $$gw$(NC)"; \
		kubectl patch gateway.gateway.networking.k8s.io "$$gw" -n "$(1)" \
			--type=merge -p '{"metadata":{"finalizers":[]}}' || true; \
		kubectl delete gateway.gateway.networking.k8s.io "$$gw" -n "$(1)" \
			--ignore-not-found=true --wait=false || true; \
	done;
endef

# $(1) - namespace
# Pre-install inventory: what already exists in the namespace and which
# controllers run there, to diagnose the helm before-hook-creation delete hang.
define dump_pre_install_state
	echo "$(CYAN)--- [diag] Gateways in $(1)$(NC)"; \
	kubectl get gateway.gateway.networking.k8s.io -n "$(1)" \
		-o custom-columns=NAME:.metadata.name,DELETING:.metadata.deletionTimestamp,FINALIZERS:.metadata.finalizers 2>/dev/null || true; \
	echo "$(CYAN)--- [diag] HTTPRoutes in $(1)$(NC)"; \
	kubectl get httproute.gateway.networking.k8s.io -n "$(1)" 2>/dev/null || true; \
	echo "$(CYAN)--- [diag] Services in $(1)$(NC)"; \
	kubectl get svc -n "$(1)" 2>/dev/null || true; \
	echo "$(CYAN)--- [diag] EnvoyFilters in $(1)$(NC)"; \
	kubectl get envoyfilter -n "$(1)" 2>/dev/null || true; \
	echo "$(CYAN)--- [diag] ConfigMaps (pod-options) in $(1)$(NC)"; \
	kubectl get configmap -n "$(1)" 2>/dev/null | grep -i pod-options || true; \
	echo "$(CYAN)--- [diag] Anything Terminating in $(1)$(NC)"; \
	kubectl get all,gateway.gateway.networking.k8s.io,httproute.gateway.networking.k8s.io -n "$(1)" \
		-o json 2>/dev/null | grep -B2 deletionTimestamp || echo "  (none)"; \
	echo "$(CYAN)--- [diag] Deployments (controllers) in $(1)$(NC)"; \
	kubectl get deploy -n "$(1)" 2>/dev/null || true; \
	echo "$(CYAN)--- [diag] GatewayClasses$(NC)"; \
	kubectl get gatewayclass 2>/dev/null || true;
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
				$(call dump_pre_install_state,$$ns) \
				$(call clear_stuck_gateways,$$ns) \
				{ $(call install_mesh_helm,$$ns) } || { $(call dump_mesh_diagnostics,$$ns) exit 1; }; \
				kubectl rollout restart deploy/core-operator -n $$ns; \
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
