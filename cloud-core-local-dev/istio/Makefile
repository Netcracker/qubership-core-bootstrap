# Istio Installation Makefile

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

# Conditional helm options
CREATE_NAMESPACE_OPTION := $(if $(filter true,$(CREATE_NAMESPACE)),--create-namespace,)

# Defaults are provided by `$(CONFIG_FILE)` (and can still be overridden via `make VAR=value`)

.PHONY: help \
	validate validate-start validate-namespace validate-crds \
	setup-repo \
	install install-gateway-api-crds install-istio-deployer enable-istio-mesh \
	uninstall uninstall-start disable-istio-mesh uninstall-istio \
	cleanup-crds cleanup-namespace \
	check-ztunnel-logs

help:
	@echo "$(BLUE)=== Istio Installation Makefile ===$(NC)"
	@echo ""
	@echo "Usage: make <target> [CONFIG_FILE=local.mk] [MESH_NAMESPACES=ns1,ns2]"	
	@echo ""
	@echo "Targets:"
	@echo "  install - Validate, deploy Istio, enable mesh"
	@echo "    validate            - Run validation steps"
	@echo "    validate-start      - Print validation start message"
	@echo "    validate-namespace  - Validate namespace (if CREATE_NAMESPACE=false)"
	@echo "    validate-crds       - Validate Gateway API CRDs (when SKIP_CRDS=true)"
	@echo "    setup-repo          - Clone/update qubership-istio-distr repository"
	@echo "    install-gateway-api-crds - Install Gateway API CRDs"
	@echo "    install-istio-deployer  - Install Istio deployer Helm chart into ISTIO_NAMESPACE"
	@echo "    enable-istio-mesh   - Enable mesh for namespaces in MESH_NAMESPACES"
	@echo ""
	@echo "  uninstall - Disable mesh, uninstall Istio, clean up CRDs/namespace"
	@echo "    uninstall-start   - Print uninstall start message"
	@echo "    disable-istio-mesh - Disable mesh (uninstall MESH_HELM_RELEASE_NAME, remove labels)"
	@echo "    uninstall-istio   - Run istioctl uninstall via deployer image"
	@echo "    cleanup-crds      - Remove Gateway API CRDs (if SKIP_CRDS=false)"
	@echo "    cleanup-namespace - Remove ISTIO_NAMESPACE (if CREATE_NAMESPACE=true)"
	@echo ""
	@echo "Variables:"
	@echo "  CONFIG_FILE - configuration file to use. Default value is \"local.mk\"."
	@echo "  MESH_NAMESPACES - comma-separated list of namespaces to enable mesh on, e.g. \"MESH_NAMESPACES=ns1,ns2\"". Default value is empty.

validate: validate-start validate-namespace validate-crds
	@echo "$(GREEN)=== Istio validation completed$(NC)";
	@echo "";

validate-start:
	@echo "$(BLUE)=== ISTIO VALIDATION START ===$(NC)";

validate-namespace:
	@if [ "$(CREATE_NAMESPACE)" = "false" ]; then \
		echo "$(BLUE)=== ISTIO NAMESPACE EXISTENCE CHECKS ===$(NC)"; \
		echo "$(CYAN)--- Checking if namespace $(ISTIO_NAMESPACE) exists...$(NC)"; \
		kubectl get namespace $(ISTIO_NAMESPACE) >/dev/null 2>&1 || (echo "$(RED)Error: Namespace $(ISTIO_NAMESPACE) not found$(NC)" && echo "$(CYAN)Please create the namespace first:$(NC)" && echo "$(CYAN)kubectl create namespace $(ISTIO_NAMESPACE)$(NC)" && exit 1); \
		echo "$(GREEN)--- Namespace $(ISTIO_NAMESPACE) found$(NC)"; \
		echo ""; \
	else \
		echo "$(CYAN)--- Skipping Istio namespace checks (CREATE_NAMESPACE=true)...$(NC)"; \
		echo ""; \
	fi

validate-crds:
	@if [ "$(SKIP_CRDS)" = "true" ]; then \
		echo "$(BLUE)=== GATEWAY API CRD CHECKS ===$(NC)"; \
		echo "$(CYAN)--- Validating Gateway API CRDs exist in cluster...$(NC)"; \
		kubectl api-resources --api-group=gateway.networking.k8s.io | grep -q gatewayclasses && \
		kubectl api-resources --api-group=gateway.networking.k8s.io | grep -q gateways && \
		kubectl api-resources --api-group=gateway.networking.k8s.io | grep -q httproutes || \
			(echo "$(RED)Error: Gateway API CRDs are missing and SKIP_CRDS=true$(NC)" && exit 1); \
		echo "$(GREEN)--- Gateway API CRDs found$(NC)"; \
		echo ""; \
	else \
		echo "$(CYAN)--- Skipping Gateway API CRD validation (SKIP_CRDS=false)...$(NC)"; \
		echo ""; \
	fi

setup-repo:
	@echo "$(YELLOW)--- Cloning Istio repository (qubership-istio-distr)...$(NC)"
	@mkdir -p $(REPOS_DIR)
	@if [ -d "$(REPOS_DIR)/qubership-istio-distr" ]; then \
		echo "Updating qubership-istio-distr..."; \
		(cd $(REPOS_DIR)/qubership-istio-distr && git pull); \
	else \
		echo "Cloning qubership-istio-distr..."; \
		git clone --depth 1 -b $(ISTIO_REPO_BRANCH) $(ISTIO_REPO_URL) $(REPOS_DIR)/qubership-istio-distr; \
	fi
	@echo ""

install-gateway-api-crds:
	@if [ "$(SKIP_CRDS)" = "false" ]; then \
		echo "$(YELLOW)--- Applying Gateway API...$(NC)"; \
		kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEWAY_API_VERSION)/$(GATEWAY_API_CHANNEL)-install.yaml; \
		echo "" ; \
		echo "$(GREEN)--- Gateway API CRDs applied$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Gateway API CRDs installation (SKIP_CRDS=true)...$(NC)"; \
		echo ""; \
	fi

install: validate validate-crds setup-repo install-gateway-api-crds install-istio-deployer enable-istio-mesh

install-istio-deployer:
	@echo "SKIP_CRDS: $(SKIP_CRDS)";
	@echo "$(BLUE)=== Deploying Istio ($(ISTIO_RELEASE_NAME) Helm chart)...$(NC)"
	@echo ""
	@helm upgrade --install $(ISTIO_RELEASE_NAME) $(REPOS_DIR)/qubership-istio-distr/helm-templates/istio-deployer \
		--namespace $(ISTIO_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		--set NAMESPACE=$(ISTIO_NAMESPACE) \
		--set meshConfig.accessLogFile="/dev/stdout" \
		$(ISTIO_HELM_EXTRA_ARGS)
	@echo ""
	@echo "$(GREEN)=== Istio deploy completed$(NC)"
	@echo ""


enable-istio-mesh:
	@if [ -z "$(MESH_NAMESPACES)" ]; then \
		echo "$(YELLOW)--- Skipping mesh enablement (MESH_NAMESPACES is empty)...$(NC)"; \
		echo ""; \
	else \
		echo "$(BLUE)=== Enabling mesh for namespaces: $(MESH_NAMESPACES)$(NC)"; \
		for ns in $$(echo "$(MESH_NAMESPACES)" | tr ',' ' '); do \
			if [ -n "$$ns" ]; then \
				echo "$(CYAN)--- Labeling namespace $$ns$(NC)"; \
				kubectl label namespace "$$ns" istio.io/dataplane-mode=ambient --overwrite; \
				kubectl label namespace "$$ns" istio.io/use-waypoint=waypoint --overwrite; \
				if [ -n "$(MESH_HELM_CHART_PATH)" ]; then \
					echo "$(CYAN)--- Installing $(MESH_HELM_RELEASE_NAME) Helm package in $$ns$(NC)"; \
					echo "MESH_HELM_EXTRA_ARGS: $(MESH_HELM_EXTRA_ARGS)$(NC)"; \
					helm dependency build $(MESH_HELM_CHART_PATH); \
					helm upgrade --install $(MESH_HELM_RELEASE_NAME) $(MESH_HELM_CHART_PATH) \
						--namespace "$$ns" \
						$(MESH_HELM_EXTRA_ARGS); \
					'$(MAKE)' -C test test NAMESPACE="$$ns" ISTIO_NAMESPACE="$(ISTIO_NAMESPACE)"; \
				else \
					echo "$(YELLOW)--- Skipping $(MESH_HELM_RELEASE_NAME) Helm package installation (MESH_HELM_CHART_PATH is empty)...$(NC)"; \
					echo ""; \
				fi; \
				echo ""; \
			fi; \
		done; \
		echo "$(GREEN)=== Mesh enabled for all namespaces$(NC)"; \
		echo ""; \
	fi

disable-istio-mesh:
	@if [ -z "$(MESH_NAMESPACES)" ]; then \
		echo "$(YELLOW)--- Skipping mesh disabling (MESH_NAMESPACES is empty)...$(NC)"; \
		echo ""; \
	else \
		echo "$(GREEN)=== Disabling mesh for namespaces: $(MESH_NAMESPACES)$(NC)"; \
		for ns in $$(echo "$(MESH_NAMESPACES)" | tr ',' ' '); do \
			if [ -n "$$ns" ]; then \
				echo "$(CYAN)--- Labeling namespace $$ns$(NC)"; \
				kubectl label namespace "$$ns" istio.io/dataplane-mode- --overwrite; \
				kubectl label namespace "$$ns" istio.io/use-waypoint- --overwrite; \
				if [ -n "$(MESH_HELM_CHART_PATH)" ]; then \
					echo "$(CYAN)--- Uninstalling $(MESH_HELM_RELEASE_NAME) Helm package in $$ns$(NC)"; \
					helm uninstall $(MESH_HELM_RELEASE_NAME) -n "$$ns" --ignore-not-found=true || true; \
				fi; \
				echo "$(GREEN)--- Mesh disabled for namespace $$ns$(NC)"; \
				echo ""; \
			fi; \
		done; \
		echo "$(GREEN)=== Mesh disabled for all namespaces$(NC)"; \
		echo ""; \
	fi

uninstall-istio:
	@echo "$(YELLOW)--- Uninstalling $(ISTIO_RELEASE_NAME) Helm chart...$(NC)";
	@helm uninstall $(ISTIO_RELEASE_NAME) -n $(ISTIO_NAMESPACE) --ignore-not-found=true || true;
	@echo "";

uninstall-start:
	@echo "$(YELLOW)--- Uninstalling Istio (disabling mesh (uninstall $(MESH_HELM_RELEASE_NAME) Helm package, remove labels), uninstall $(ISTIO_RELEASE_NAME) Helm chart)...$(NC)";
	@echo ""

uninstall: uninstall-start disable-istio-mesh uninstall-istio cleanup-crds cleanup-namespace
	@echo "$(GREEN)=== Istio uninstall completed$(NC)";
	@echo ""

cleanup-crds:
	@if [ "$(SKIP_CRDS)" = "false" ]; then \
		echo "$(YELLOW)--- Cleaning up Gateway API CRDs...$(NC)"; \
		kubectl delete -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEWAY_API_VERSION)/$(GATEWAY_API_CHANNEL)-install.yaml --ignore-not-found=true; \
		echo "$(GREEN)=== Gateway API CRDs cleaned up$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Gateway API CRDs cleanup (SKIP_CRDS=true)...$(NC)"; \
		echo ""; \
	fi

cleanup-namespace:
	@if [ "$(CREATE_NAMESPACE)" = "true" ]; then \
		echo "$(YELLOW)--- Cleaning up namespace...$(NC)"; \
		kubectl delete namespace $(ISTIO_NAMESPACE) --ignore-not-found=true; \
		echo "$(GREEN)=== Namespace cleaned up$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping namespace cleanup (CREATE_NAMESPACE=false)...$(NC)"; \
		echo ""; \
	fi
