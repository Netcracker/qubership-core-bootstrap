# Istio Distribution Installation Makefile

# Include configuration
CONFIG_FILE ?= local.mk
include $(CONFIG_FILE)

# Repository directory
REPOS_DIR ?= ./.tmp

# Where the chart comes from, see ISTIO_CHART_SOURCE in the config file. The two
# sources use separate directories so switching between them never overwrites a
# working git clone with an extracted image, or the other way round.
ifeq ($(ISTIO_CHART_SOURCE),git)
ISTIO_CHART_DIR := $(REPOS_DIR)/qubership-istio/helm-templates/qubership-istio
else
ISTIO_CHART_DIR := $(REPOS_DIR)/qubership-istio-transfer/helm-templates/qubership-istio
endif
# Tag of the transfer image: explicit ISTIO_IMAGE_TAG wins, otherwise the one
# qubership-istio CI publishes for ISTIO_REPO_BRANCH.
ISTIO_IMAGE_TAG := $(if $(ISTIO_IMAGE_TAG),$(ISTIO_IMAGE_TAG),$(if $(filter main,$(ISTIO_REPO_BRANCH)),$(ISTIO_MINOR_VERSION)-latest,$(ISTIO_MINOR_VERSION)-$(subst /,_,$(ISTIO_REPO_BRANCH))))

# Define colors for output
YELLOW := \033[1;33m
GREEN := \033[1;32m
RED := \033[0;31m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m

# Conditional helm options
CREATE_NAMESPACE_OPTION := $(if $(filter true,$(CREATE_NAMESPACE)),--create-namespace,)

.PHONY: help \
	validate validate-start validate-namespace validate-crds \
	setup-repo setup-chart-git setup-chart-image \
	install install-gateway-api-crds deploy-qubership-istio \
	uninstall uninstall-start uninstall-istio \
	cleanup-crds cleanup-namespace

help:
	@echo "$(BLUE)=== Istio Distribution Makefile ===$(NC)"
	@echo ""
	@echo "Usage: make <target> [CONFIG_FILE=local.mk]"
	@echo ""
	@echo "Targets:"
	@echo "  install                  - Validate, fetch the chart, install Gateway API CRDs, deploy Istio"
	@echo "    validate               - Run validation steps"
	@echo "    validate-start         - Print validation start message"
	@echo "    validate-namespace     - Validate ISTIO_NAMESPACE exists (if CREATE_NAMESPACE=false)"
	@echo "    validate-crds          - Validate Gateway API CRDs exist (if SKIP_CRDS=true)"
	@echo "    setup-repo             - Fetch the qubership-istio chart (setup-chart-image or setup-chart-git, per ISTIO_CHART_SOURCE)"
	@echo "    setup-chart-image      - Extract the chart with vendored sub-charts from the qubership-istio-transfer image"
	@echo "    setup-chart-git        - Clone/update the qubership-istio repository"
	@echo "    install-gateway-api-crds - Install Gateway API CRDs"
	@echo "    deploy-qubership-istio - Install qubership-istio Helm chart into ISTIO_NAMESPACE"
	@echo ""
	@echo "  uninstall                - Uninstall Istio and clean up CRDs/namespace"
	@echo "    uninstall-start        - Print uninstall start message"
	@echo "    uninstall-istio        - Uninstall qubership-istio Helm chart"
	@echo "    cleanup-crds           - Remove Gateway API CRDs (if SKIP_CRDS=false)"
	@echo "    cleanup-namespace      - Remove ISTIO_NAMESPACE (if CREATE_NAMESPACE=true)"
	@echo ""
	@echo "Variables:"
	@echo "  CONFIG_FILE          - Configuration file to use (default: local.mk)"
	@echo "  ISTIO_NAMESPACE      - Namespace to install Istio into (default: istio-system)"
	@echo "  CREATE_NAMESPACE     - Create ISTIO_NAMESPACE if missing (default: true)"
	@echo "  SKIP_CRDS            - Skip Gateway API CRD installation (default: false)"
	@echo "  ISTIO_CHART_SOURCE   - image: chart from the qubership-istio-transfer image (default); git: clone qubership-istio"
	@echo "  ISTIO_IMAGE          - Transfer image (default: ghcr.io/netcracker/qubership-istio-transfer)"
	@echo "  ISTIO_IMAGE_TAG      - Transfer image tag (default: derived from ISTIO_MINOR_VERSION and ISTIO_REPO_BRANCH)"
	@echo "  ISTIO_MINOR_VERSION  - Minor version the image tags are prefixed with (default: 1.30)"
	@echo "  ISTIO_REPO_URL       - Git URL for qubership-istio (ISTIO_CHART_SOURCE=git)"
	@echo "  ISTIO_REPO_BRANCH    - Branch to use (default: main); also selects the image tag"
	@echo "  ISTIO_HELM_EXTRA_ARGS - Extra helm args for qubership-istio chart"
	@echo "  GATEWAY_API_VERSION  - Gateway API version to install (default: v1.4.0)"
	@echo "  GATEWAY_API_CHANNEL  - Gateway API channel (default: standard)"

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

ifeq ($(ISTIO_CHART_SOURCE),git)
setup-repo: setup-chart-git
else
setup-repo: setup-chart-image
endif

# The transfer image is FROM scratch and carries only /helm-templates, with the
# Istio sub-charts already in charts/, so a container is created but never run.
setup-chart-image:
	@echo "$(YELLOW)--- Fetching qubership-istio chart from $(ISTIO_IMAGE):$(ISTIO_IMAGE_TAG)...$(NC)"
	@command -v docker >/dev/null 2>&1 || (echo "$(RED)Error: docker is required to extract the chart from the image. Install it, or set ISTIO_CHART_SOURCE=git to clone the repository instead$(NC)" && exit 1)
	@docker pull $(ISTIO_IMAGE):$(ISTIO_IMAGE_TAG)
	@rm -rf $(REPOS_DIR)/qubership-istio-transfer
	@mkdir -p $(REPOS_DIR)/qubership-istio-transfer
	@CID=$$(docker create $(ISTIO_IMAGE):$(ISTIO_IMAGE_TAG) sh) && \
		docker cp $$CID:/helm-templates $(REPOS_DIR)/qubership-istio-transfer/helm-templates && \
		docker rm $$CID >/dev/null
	@echo "$(CYAN)--- Chart extracted to $(ISTIO_CHART_DIR)$(NC)"
	@if [ -f "$(ISTIO_CHART_DIR)/effective-images.yaml" ]; then \
		echo "$(CYAN)--- Istio images it installs:$(NC)"; \
		sed 's/^/    /' "$(ISTIO_CHART_DIR)/effective-images.yaml"; \
	fi
	@echo ""

setup-chart-git:
	@echo "$(YELLOW)--- Setting up qubership-istio repository...$(NC)"
	@mkdir -p $(REPOS_DIR)
	@if [ -d "$(REPOS_DIR)/qubership-istio" ]; then \
		CURRENT_BRANCH=$$(cd $(REPOS_DIR)/qubership-istio && git rev-parse --abbrev-ref HEAD); \
		if [ "$$CURRENT_BRANCH" != "$(ISTIO_REPO_BRANCH)" ]; then \
			echo "$(CYAN)--- Switching qubership-istio from $$CURRENT_BRANCH to $(ISTIO_REPO_BRANCH)...$(NC)"; \
			(cd $(REPOS_DIR)/qubership-istio && git fetch origin && git checkout $(ISTIO_REPO_BRANCH) && git pull); \
		else \
			echo "$(CYAN)--- Updating qubership-istio (branch: $(ISTIO_REPO_BRANCH))...$(NC)"; \
			(cd $(REPOS_DIR)/qubership-istio && git pull); \
		fi; \
	else \
		echo "$(CYAN)--- Cloning qubership-istio (branch: $(ISTIO_REPO_BRANCH))...$(NC)"; \
		git clone -b $(ISTIO_REPO_BRANCH) $(ISTIO_REPO_URL) $(REPOS_DIR)/qubership-istio; \
	fi
	@echo ""

install-gateway-api-crds:
	@if [ "$(SKIP_CRDS)" = "false" ]; then \
		echo "$(YELLOW)--- Applying Gateway API CRDs ($(GATEWAY_API_VERSION) $(GATEWAY_API_CHANNEL))...$(NC)"; \
		kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEWAY_API_VERSION)/$(GATEWAY_API_CHANNEL)-install.yaml; \
		echo ""; \
		echo "$(GREEN)--- Gateway API CRDs applied$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Gateway API CRDs installation (SKIP_CRDS=true)...$(NC)"; \
		echo ""; \
	fi

deploy-qubership-istio:
	@echo "$(BLUE)=== Deploying Istio ($(ISTIO_RELEASE_NAME) Helm chart from $(ISTIO_CHART_DIR))...$(NC)"
	@echo ""
	@if [ "$(ISTIO_CHART_SOURCE)" = "git" ]; then \
		echo "$(CYAN)--- Resolving chart dependencies from the upstream Istio chart repository...$(NC)"; \
		helm dependency update $(ISTIO_CHART_DIR); \
	fi
	@helm upgrade --install $(ISTIO_RELEASE_NAME) $(ISTIO_CHART_DIR) \
		--namespace $(ISTIO_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		--set global.istioNamespace=$(ISTIO_NAMESPACE) \
		--set MONITORING_ENABLED=$(MONITORING_ENABLED) \
		--force-conflicts \
		$(ISTIO_HELM_EXTRA_ARGS)
	@echo ""
	@echo "$(GREEN)=== Istio deploy completed$(NC)"
	@echo ""

install: validate setup-repo install-gateway-api-crds deploy-qubership-istio
	@echo "$(GREEN)=== Istio installation completed$(NC)"
	@echo ""

uninstall-istio:
	@echo "$(YELLOW)--- Uninstalling $(ISTIO_RELEASE_NAME) Helm chart...$(NC)"
	@helm uninstall $(ISTIO_RELEASE_NAME) -n $(ISTIO_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-start:
	@echo "$(YELLOW)--- Uninstalling Istio ($(ISTIO_RELEASE_NAME) Helm chart)...$(NC)"
	@echo ""

uninstall: uninstall-start uninstall-istio cleanup-crds cleanup-namespace
	@echo "$(GREEN)=== Istio uninstall completed$(NC)"
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
		echo "$(YELLOW)--- Cleaning up namespace $(ISTIO_NAMESPACE)...$(NC)"; \
		kubectl delete namespace $(ISTIO_NAMESPACE) --ignore-not-found=true; \
		echo "$(GREEN)=== Namespace cleaned up$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping namespace cleanup (CREATE_NAMESPACE=false)...$(NC)"; \
		echo ""; \
	fi
