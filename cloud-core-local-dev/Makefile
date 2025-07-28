# =============================================================================
# PHONY TARGETS
# =============================================================================

# Main targets
.PHONY: all install help

# Installation targets
.PHONY: deploy deploy-dependencies deploy-core \
	clone-repos template-values-files \
	deploy-consul deploy-consul-old deploy-consul-print-start install-consul wait-for-consul check-consul-connectivity \
	deploy-monitoring deploy-metrics-server \
	deploy-dbaas \
	deploy-maas \
	prepare-core-bootstrap prepare-core-bootstrap-print-start deploy-cloud-core-configuration \
	deploy-cloud-core-components deploy-cloud-core-print-start \
	apply-facade-operator-crds deploy-facade-operator \
	deploy-ingress-gateway \
	deploy-site-management \
	deploy-paas-mediation \
	deploy-control-plane \
	deploy-dbaas-agent \
	apply-core-operator-crds deploy-core-operator \
	deploy-config-server

# Uninstallation targets
.PHONY: uninstall uninstall-core uninstall-dependencies \
	uninstall-cloud-core-components uninstall-cloud-core-configuration \
	uninstall-consul uninstall-monitoring uninstall-metrics-server \
	uninstall-dbaas \
	uninstall-maas \
	uninstall-control-plane-db-credentials \
	uninstall-facade-operator uninstall-ingress-gateway uninstall-site-management \
	uninstall-paas-mediation uninstall-control-plane uninstall-dbaas-agent \
	uninstall-core-operator uninstall-config-server \
	cleanup-crds cleanup-namespaces

# Validation and utility targets
.PHONY: validate validate-cli validate-crd \
	ensure-core-namespace validate-consul-namespace \
	change-consul-values clone-consul-repo \
	mesh-smoke-test cleanup-mesh-test

# =============================================================================
# CONFIGURATION
# =============================================================================

# Load configuration from .mk file
CONFIG_FILE ?= local.mk
include $(CONFIG_FILE)

# Color definitions
YELLOW := \033[1;33m
GREEN := \033[1;32m
RED := \033[0;31m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m

CREATE_NAMESPACE_OPTION ?= $(if $(filter true,$(CREATE_NAMESPACE)), --create-namespace, )

ifeq ($(DEBUG),true)
  HELM_DEBUG_FLAGS := --debug --dry-run
else
  HELM_DEBUG_FLAGS :=
endif

# =============================================================================
# HELP
# =============================================================================

# Help target
help:
	@echo "$(BLUE)=== Cloud Core Local Development Makefile ===$(NC)"
	@echo ""
	@echo "$(CYAN)USAGE:$(NC)"
	@echo "  make [target] [CONFIG_FILE=local.mk] [DEBUG=true]"
	@echo ""
	@echo "$(CYAN)CONFIGURATION VARIABLES:$(NC)"
	@echo "  CONFIG_FILE=local.mk    # Configuration file (default: local.mk)"
	@echo "  DEBUG=true              # Enable debug mode with --debug --dry-run"
	@echo ""
	@echo "$(CYAN)MAIN TARGETS:$(NC)"
	@echo "  install                 # Complete installation: validate prerequisites, deploy Cloud Core and dependencies, mesh connectivity smoke test"
	@echo "  deploy                  # Deploy Cloud Core and dependencies"
	@echo "  uninstall               # Complete uninstall, CRDs and namespaces cleanup"
	@echo ""
	@echo "$(CYAN)INSTALLATION TARGETS:$(NC)"
	@echo "  deploy-core             # Deploy Cloud Core bootstrap configuration and components"
	@echo "  deploy-dependencies     # Deploy metrics-server, monitoring, consul, dbaas"
	@echo "  deploy-consul           # Deploy Consul (if INSTALL_CONSUL=true)"
	@echo "  deploy-monitoring       # Deploy monitoring operator (if INSTALL_MONITORING=true)"
	@echo "  deploy-metrics-server   # Deploy metrics server (if INSTALL_METRICS_SERVER=true)"
	@echo "  deploy-dbaas            # Deploy DBaaS (if INSTALL_DBAAS=true)"
	@echo "  deploy-maas             # Deploy MaaS (if INSTALL_MAAS=true)"
	@echo ""
	@echo "$(CYAN)VALIDATION TARGETS:$(NC)"
	@echo "  validate                # Validate CLI tools and CRDs"
	@echo "  validate-cli            # Check helm, git, kubectl availability"
	@echo "  validate-crd            # Check required CRDs"
	@echo "  check-consul-connectivity # Test Consul connectivity (if INSTALL_CONSUL=true)"
	@echo ""
	@echo "$(CYAN)TESTING TARGETS:$(NC)"
	@echo "  mesh-smoke-test         # Run mesh connectivity test"
	@echo "  cleanup-mesh-test       # Clean up mesh test resources"
	@echo ""
	@echo "$(CYAN)UNINSTALLATION TARGETS:$(NC)"
	@echo "  uninstall-core          # Uninstall Cloud Core bootstrap configuration and components"
	@echo "  uninstall-dependencies  # Uninstall dependencies"
	@echo "  uninstall-consul        # Uninstall Consul (if INSTALL_CONSUL=true)"
	@echo "  uninstall-monitoring    # Uninstall monitoring (if INSTALL_MONITORING=true)"
	@echo "  uninstall-metrics-server # Uninstall metrics server (if INSTALL_METRICS_SERVER=true)"
	@echo "  uninstall-dbaas         # Uninstall DBaaS (if INSTALL_DBAAS=true)"
	@echo "  uninstall-maas          # Uninstall MaaS (if INSTALL_MAAS=true)"
	@echo "  cleanup-crds            # Remove all CRDs (if INSTALL_CRDS=true)"
	@echo "  cleanup-namespaces      # Remove all namespaces (if CREATE_NAMESPACE=true)"
	@echo ""
	@echo "$(CYAN)UTILITY TARGETS:$(NC)"
	@echo "  clone-repos             # Clone/update required repositories"
	@echo "  template-values-files   # Generate values files from config"
	@echo "  ensure-core-namespace   # Ensure core namespace exists"
	@echo ""
	@echo "$(CYAN)CONFIGURATION FILES:$(NC)"
	@echo "  local.mk                # Local development configuration (default)"
	@echo "  aws.mk                  # AWS environment configuration"
	@echo ""
	@echo "$(YELLOW)For more information, see README.md$(NC)"
	@echo ""

# =============================================================================
# INSTALLATION
# =============================================================================

install: validate clone-repos template-values-files deploy mesh-smoke-test deploy-maas

deploy: deploy-dependencies deploy-core

deploy-dependencies: deploy-metrics-server deploy-monitoring deploy-consul deploy-dbaas
	@echo "$(GREEN)=== Cloud Core dependencies deployment completed$(NC)"
	@echo ""

deploy-core: ensure-core-namespace prepare-core-bootstrap deploy-cloud-core-components
	@echo "$(GREEN)=== Cloud Core deployment completed$(NC)"
	@echo ""

#-------------------------------Validate prerequisites-------------------------------------------------------------

validate: validate-cli validate-crd

validate-cli:
	@echo "$(BLUE)=== CLI PREREQUISITE CHECKS ===$(NC)"
	@echo "$(CYAN)--- Checking if helm is available...$(NC)"
	@command -v helm >/dev/null 2>&1 || (echo "$(RED)Error: helm is not installed or not in PATH$(NC)" && echo "$(CYAN)Please install helm: https://helm.sh/docs/intro/install/$(NC)" && exit 1)
	@echo "$(GREEN)--- helm found: $(shell helm version --short)$(NC)"
	@echo "$(CYAN)--- Checking if git is available...$(NC)"
	@command -v git >/dev/null 2>&1 || (echo "$(RED)Error: git is not installed or not in PATH$(NC)" && echo "$(CYAN)Please install git: https://git-scm.com/downloads$(NC)" && exit 1)
	@echo "$(GREEN)--- git found: $(shell git --version)$(NC)"
	@echo "$(CYAN)--- Checking if kubectl is available...$(NC)"
	@command -v kubectl >/dev/null 2>&1 || (echo "$(RED)Error: kubectl is not installed or not in PATH$(NC)" && echo "$(CYAN)Please install kubectl: https://kubernetes.io/docs/tasks/tools/$(NC)" && exit 1)
	@echo "$(GREEN)--- kubectl found: $(shell kubectl version --client)$(NC)"
	@echo "$(CYAN)--- Checking kubectl cluster connection...$(NC)"
	@kubectl cluster-info >/dev/null 2>&1 || (echo "$(RED)Error: kubectl cannot connect to cluster$(NC)" && echo "$(CYAN)Please check your kubeconfig and cluster status$(NC)" && exit 1)
	@echo "$(GREEN)--- kubectl cluster connection successful$(NC)"
	@echo ""

validate-crd:
	@echo "$(BLUE)=== CRD PREREQUISITE CHECKS ===$(NC)"
	@if [ "$(INSTALL_CRDS)" = "false" ]; then \
		echo "$(CYAN)--- Checking Facade Operator CRDs (INSTALL_CRDS=false)...$(NC)"; \
		$(call check_crd_existence,facadeservices.qubership.org,facadeservices,kubectl apply -f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/new-crd/facade-service-crd-v1.yaml); \
		$(call check_crd_existence,gateways.core.qubership.org,gateways,kubectl apply -f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/new-crd/gateway-crd-v1.yaml); \
		echo "$(CYAN)--- Checking Core Operator CRDs (INSTALL_CRDS=false)...$(NC)"; \
		$(call check_crd_existence,securities.core.qubership.org,securities,kubectl apply -f ./repos/qubership-core-core-operator/helm-templates/core-operator/new-crd/security.yaml); \
		$(call check_crd_existence,meshes.core.qubership.org,meshes,kubectl apply -f ./repos/qubership-core-core-operator/helm-templates/core-operator/new-crd/mesh.yaml); \
		$(call check_crd_existence,dbaases.core.qubership.org,dbaases,kubectl apply -f ./repos/qubership-core-core-operator/helm-templates/core-operator/new-crd/dbaas.yaml); \
		$(call check_crd_existence,maases.core.qubership.org,maases,kubectl apply -f ./repos/qubership-core-core-operator/helm-templates/core-operator/new-crd/maas.yaml); \
		$(call check_crd_existence,composites.core.qubership.org,composites,kubectl apply -f ./repos/qubership-core-core-operator/helm-templates/core-operator/new-crd/composite.yaml); \
	else \
		echo "$(CYAN)--- Skipping additional CRD checks (INSTALL_CRDS=true, CRDs will be installed automatically)...$(NC)"; \
	fi
	@echo ""

#-------------------core namespace-----------------------------------------------------------------------------

# Core Namespace required for Consul connectivity check pod deployment
ensure-core-namespace:
	@if [ "$(CREATE_NAMESPACE)" = "true" ]; then \
		echo "$(GREEN)=== Ensuring Core namespace exists...$(NC)"; \
		echo ""; \
		$(call ensure_namespace_exists,$(CORE_NAMESPACE)); \
		echo "$(GREEN)=== Core namespace check completed$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Checking Core namespace existence (CREATE_NAMESPACE=false)...$(NC)"; \
		echo ""; \
		$(call check_namespace_existence,$(CORE_NAMESPACE)); \
		echo "$(GREEN)=== Core namespace check completed$(NC)"; \
		echo ""; \
	fi

# ---------------------- clone repos ---------------------------------------------

clone-repos:
	@echo "$(YELLOW)--- Cloning required repositories...$(NC)"
	mkdir -p repos
	@$(call clone_or_update_repo,qubership-monitoring-operator,https://github.com/Netcracker/qubership-monitoring-operator.git,main)
	@$(call clone_or_update_repo,qubership-core-paas-mediation,https://github.com/Netcracker/qubership-core-paas-mediation.git,feature/local-deployment)
	@$(call clone_or_update_repo,qubership-core-control-plane,https://github.com/Netcracker/qubership-core-control-plane.git,feature/local-deployment)
	@$(call clone_or_update_repo,qubership-core-ingress-gateway,https://github.com/Netcracker/qubership-core-ingress-gateway.git,feature/local-deployment)
	@$(call clone_or_update_repo,qubership-core-site-management,https://github.com/Netcracker/qubership-core-site-management.git,feature/local-deployment)
	@$(call clone_or_update_repo,qubership-core-facade-operator,https://github.com/Netcracker/qubership-core-facade-operator.git,feature/local-deployment)
	@$(call clone_or_update_repo,qubership-core-dbaas-agent,https://github.com/Netcracker/qubership-core-dbaas-agent.git,feature/local-deployment)
	@$(call clone_or_update_repo,qubership-core-core-operator,https://github.com/Netcracker/qubership-core-core-operator.git,feature/local-deployment)
	@$(call clone_or_update_repo,qubership-core-config-server,https://github.com/Netcracker/qubership-core-config-server.git,feature/local-deployment)
	@echo ""

#---------------------------- values templating ---------------------------------------------

template-values-files:
	@$(call export_config_vars)
	@echo "core-values.yaml: $(shell cat ./core-values.yaml)"

# -------------------------- consul -----------------------

deploy-consul-old:
	@echo "$(GREEN)=== Deploying Consul...$(NC)"
	@echo ""
	helm upgrade --install --values ./consul-chart-3/values.yaml consul ./consul-chart-3/ --namespace consul $$(CREATE_NAMESPACE_OPTION)
	@echo "$(GREEN)=== Consul is deployed$(NC)"
	@echo ""

deploy-consul: deploy-consul-print-start validate-consul-namespace ensure-core-namespace clone-consul-repo change-consul-values wait-for-consul check-consul-connectivity
	@echo "$(GREEN)=== Consul setup completed...$(NC)"
	@echo ""

deploy-consul-print-start:
	@echo "$(GREEN)=== Starting Consul setup...$(NC)"
	@echo ""

validate-consul-namespace:
	@if [ "$(CREATE_NAMESPACE)" = "false" ]; then \
		echo "$(BLUE)=== CONSUL NAMESPACE EXISTENCE CHECKS ===$(NC)"; \
		$(call check_namespace_existence,$(CONSUL_NAMESPACE)); \
	else \
		echo "$(CYAN)--- Skipping Consul namespace checks (CREATE_NAMESPACE=true)...$(NC)"; \
	fi
	@echo ""

clone-consul-repo:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Cloning consul repository...$(NC)"; \
		$(call clone_or_update_repo,consul-k8s,https://github.com/hashicorp/consul-k8s,main); \
	else \
		echo "$(YELLOW)--- Skipping Consul repository cloning (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

change-consul-values:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Updating Consul Helm chart security context...$(NC)"; \
		if [ -f ./repos/consul-k8s/charts/consul/templates/_helpers.tpl ]; then \
			if command -v sed >/dev/null 2>&1; then \
				if sed --version 2>/dev/null | grep -q "GNU sed"; then \
					sed -i 's/runAsNonRoot: true/runAsNonRoot: false/g' ./repos/consul-k8s/charts/consul/templates/_helpers.tpl; \
					sed -i 's/runAsUser: 100/runAsUser: 0/g' ./repos/consul-k8s/charts/consul/templates/_helpers.tpl; \
				else \
					sed -i '' 's/runAsNonRoot: true/runAsNonRoot: false/g' ./repos/consul-k8s/charts/consul/templates/_helpers.tpl; \
					sed -i '' 's/runAsUser: 100/runAsUser: 0/g' ./repos/consul-k8s/charts/consul/templates/_helpers.tpl; \
				fi; \
			else \
				echo "Warning: sed not found, using alternative method"; \
				cp ./repos/consul-k8s/charts/consul/templates/_helpers.tpl ./repos/consul-k8s/charts/consul/templates/_helpers.tpl.bak; \
				cat ./repos/consul-k8s/charts/consul/templates/_helpers.tpl.bak | sed 's/runAsNonRoot: true/runAsNonRoot: false/g' | sed 's/runAsUser: 100/runAsUser: 0/g' > ./repos/consul-k8s/charts/consul/templates/_helpers.tpl; \
			fi; \
			echo "Security context updated in _helpers.tpl"; \
		else \
			echo "Error: _helpers.tpl not found"; \
			exit 1; \
		fi; \
	else \
		echo "$(YELLOW)--- Skipping Consul values update (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

install-consul:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Installing consul...$(NC)"; \
		helm upgrade --install --namespace $(CONSUL_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
			consul ./repos/consul-k8s/charts/consul/ \
			-f ./repos/consul-k8s/charts/consul/values.yaml \
			--set server.securityContext.runAsNonRoot=false \
			--set server.securityContext.runAsUser=0 \
			--set connectInject.enabled=false; \
	else \
		echo "$(YELLOW)--- Skipping Consul installation (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

wait-for-consul: install-consul
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Waiting for Consul pods to be ready (timeout: 5 minutes)...$(NC)"; \
		TIMEOUT=300; \
		START_TIME=$$(date +%s); \
		while true; do \
			CURRENT_TIME=$$(date +%s); \
			ELAPSED_TIME=$$((CURRENT_TIME - START_TIME)); \
			if [ $$ELAPSED_TIME -ge $$TIMEOUT ]; then \
				echo "Timeout reached after 5 minutes. Consul pods are not ready."; \
				exit 1; \
			fi; \
			RUNNING_CONSUL_PODS=$$(kubectl -n $(CONSUL_NAMESPACE) get pods --selector=app=consul --field-selector=status.phase=Running -o name | wc -l); \
			if [ "$$RUNNING_CONSUL_PODS" -ge 1 ]; then \
				echo "Consul pods are running ($$RUNNING_CONSUL_PODS)!"; \
				break; \
			fi; \
			echo "Waiting for Consul pods to be ready... ($$RUNNING_CONSUL_PODS) - $$((TIMEOUT - ELAPSED_TIME))s remaining"; \
			sleep 10; \
		done; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Consul wait (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

check-consul-connectivity:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
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

# ----------------- monitoring --------------------------

deploy-monitoring:
	@if [ "$(INSTALL_MONITORING)" = "true" ]; then \
		echo "$(GREEN)=== Deploying Monitoring...$(NC)"; \
		echo ""; \
		if [ "$(CREATE_NAMESPACE)" = "false" ]; then \
			$(call check_namespace_existence,$(MONITORING_NAMESPACE)); \
			echo ""; \
		fi; \
		helm upgrade --install --namespace $(MONITORING_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
			monitoring-operator ./repos/qubership-monitoring-operator/charts/qubership-monitoring-operator \
			-f ./repos/qubership-monitoring-operator/charts/qubership-monitoring-operator/values.yaml; \
		echo "$(GREEN)=== Monitoring is deployed$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Monitoring deployment (MONITORING_ENABLED=false)...$(NC)"; \
		echo ""; \
	fi

# ------------------- metrics-server -------------------------------

deploy-metrics-server:
	@if [ "$(INSTALL_METRICS_SERVER)" = "true" ]; then \
		echo "$(GREEN)=== Deploying Metrics Server...$(NC)"; \
		echo ""; \
		kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml; \
		kubectl patch deployment metrics-server -n kube-system --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]'; \
		echo "$(GREEN)=== Metrics Server is deployed$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Metrics Server deployment (INSTALL_METRICS_SERVER=false)...$(NC)"; \
		echo ""; \
	fi

# --------------------- bootstrap --------------------

prepare-core-bootstrap: prepare-core-bootstrap-print-start deploy-cloud-core-configuration
	@echo "$(GREEN)=== Bootstrap preparation completed$(NC)"
	@echo ""

prepare-core-bootstrap-print-start:
	@echo "$(GREEN)=== Starting bootstrap preparation...$(NC)"
	@echo ""

deploy-cloud-core-configuration:
	@echo "$(YELLOW)--- Deploying cloud-core-configuration$(NC)"
	helm upgrade --install -n $(CORE_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		cloud-core-app-chart ../cloud-core-configuration/helm-charts \
		-f ./core-values.yaml \
		--set CONSUL_ENABLED=$(CORE_CONFIG_CONSUL_ENABLED) \
		--set NAMESPACE=$(CORE_NAMESPACE) \
		--set CORE_BOOTSTRAP_IMAGE=$(CORE_BOOTSTRAP_IMAGE) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

# ------------------ cloud core ---------------------------

deploy-cloud-core-components: deploy-cloud-core-print-start deploy-facade-operator deploy-ingress-gateway deploy-control-plane deploy-paas-mediation deploy-dbaas-agent deploy-core-operator deploy-config-server deploy-site-management
	@echo "$(GREEN)=== Cloud Core components deployment completed$(NC)"
	@echo ""

deploy-cloud-core-print-start:
	@echo "$(GREEN)=== Starting Cloud Core installation...$(NC)"
	@echo ""

apply-facade-operator-crds:
	@if [ "$(INSTALL_CRDS)" = "true" ]; then \
		echo "$(YELLOW)--- Create Facade Operator CRDs...$(NC)"; \
		if ! kubectl get crd facadeservices.qubership.org >/dev/null 2>&1; then \
			echo "Creating facadeservices.qubership.org CRD..."; \
			kubectl apply -f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/new-crd/facade-service-crd-v1.yaml; \
		else \
			echo "facadeservices.qubership.org CRD already exists"; \
		fi; \
		if ! kubectl get crd gateways.core.qubership.org >/dev/null 2>&1; then \
			echo "Creating gateways.core.qubership.org CRD..."; \
			kubectl apply -f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/new-crd/gateway-crd-v1.yaml; \
		else \
			echo "gateways.core.qubership.org CRD already exists"; \
		fi; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Facade Operator CRDs installation (INSTALL_CRDS=false)...$(NC)"; \
		echo ""; \
	fi

deploy-facade-operator: apply-facade-operator-crds
	@echo "$(YELLOW)--- Setting up Facade Operator...$(NC)"
	@$(call update_helm_dependencies,repos/qubership-core-facade-operator/helm-templates/facade-operator,)
	helm upgrade --install facade-operator ./repos/qubership-core-facade-operator/helm-templates/facade-operator -n $(CORE_NAMESPACE) \
		-f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set TAG=$(FACADE_OPERATOR_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-ingress-gateway: 
	@echo "$(YELLOW)--- Setting up Ingress Gateway...$(NC)"
	@$(call update_helm_dependencies,repos/qubership-core-ingress-gateway/helm-templates/frontend-gateway,)
	helm upgrade --install ingress-gateway ./repos/qubership-core-ingress-gateway/helm-templates/frontend-gateway -n $(CORE_NAMESPACE) \
		-f ./repos/qubership-core-ingress-gateway/helm-templates/frontend-gateway/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set CLOUD_PUBLIC_HOST=$(INGRESS_GATEWAY_CLOUD_PUBLIC_HOST) \
		--set CLOUD_PRIVATE_HOST=$(INGRESS_GATEWAY_CLOUD_PRIVATE_HOST) \
		--set TAG=$(INGRESS_GATEWAY_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-site-management: 
	@echo "$(YELLOW)--- Setting up Site Management...$(NC)"
	@$(call update_helm_dependencies,repos/qubership-core-site-management/helm-templates/site-management,)
	helm upgrade --install site-management ./repos/qubership-core-site-management/helm-templates/site-management -n $(CORE_NAMESPACE) \
		-f ./repos/qubership-core-site-management/helm-templates/site-management/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set TAG=$(SITE_MANAGEMENT_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-paas-mediation: 
	@echo "$(YELLOW)--- Setting up PaaS Mediation...$(NC)"
	@$(call update_helm_dependencies,repos/qubership-core-paas-mediation/helm-templates/paas-mediation,)
	helm upgrade --install -n $(CORE_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		paas-mediation ./repos/qubership-core-paas-mediation/helm-templates/paas-mediation \
		-f ./repos/qubership-core-paas-mediation/helm-templates/paas-mediation/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set TAG=$(PAAS_MEDIATION_TAG) \
		--set CONSUL_ENABLED=false \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-control-plane: 
	@echo "$(YELLOW)--- Setting up Control Plane...$(NC)"
	@$(call update_helm_dependencies,repos/qubership-core-control-plane/helm-templates/control-plane,)
	helm upgrade --install -n $(CORE_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		control-plane ./repos/qubership-core-control-plane/helm-templates/control-plane \
		-f ./repos/qubership-core-control-plane/helm-templates/control-plane/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set TAG=$(CONTROL_PLANE_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-dbaas-agent:
	@echo "$(YELLOW)--- Setting up DBaaS agent...$(NC)"
	@$(call update_helm_dependencies,repos/qubership-core-dbaas-agent/helm-templates/dbaas-agent,)
	helm upgrade --install dbaas-agent ./repos/qubership-core-dbaas-agent/helm-templates/dbaas-agent -n $(CORE_NAMESPACE) \
		-f ./repos/qubership-core-dbaas-agent/helm-templates/dbaas-agent/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set TAG=$(DBAAS_AGENT_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

apply-core-operator-crds:
	@if [ "$(INSTALL_CRDS)" = "true" ]; then \
		echo "$(YELLOW)--- Applying Core Operator CRDs...$(NC)"; \
		kubectl apply -f ./repos/qubership-core-core-operator/helm-templates/core-operator/new-crd; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Core Operator CRDs installation (INSTALL_CRDS=false)...$(NC)"; \
		echo ""; \
	fi

deploy-core-operator: apply-core-operator-crds
	@echo "$(YELLOW)--- Setting up Core Operator...$(NC)"
	@$(call update_helm_dependencies,repos/qubership-core-core-operator/helm-templates/core-operator,)
	helm upgrade --install core-operator ./repos/qubership-core-core-operator/helm-templates/core-operator -n $(CORE_NAMESPACE) \
		-f ./repos/qubership-core-core-operator/helm-templates/core-operator/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set IMAGE_REPOSITORY=$(CORE_OPERATOR_IMAGE_REPOSITORY) \
		--set TAG=$(CORE_OPERATOR_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-config-server: 
	@echo "$(YELLOW)--- Setting up Config Server...$(NC)"
	@$(call update_helm_dependencies,repos/qubership-core-config-server/helm-templates/config-server,)
	helm upgrade --install config-server ./repos/qubership-core-config-server/helm-templates/config-server -n $(CORE_NAMESPACE) \
		-f ./repos/qubership-core-config-server/helm-templates/config-server/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set IMAGE_REPOSITORY=$(CONFIG_SERVER_IMAGE_REPOSITORY) \
		--set TAG=$(CONFIG_SERVER_TAG) \
		--set CONSUL_ENABLED=$(CONFIG_SERVER_CONSUL_ENABLED) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

# =============================================================================
# UNINSTALL
# =============================================================================

uninstall: uninstall-maas uninstall-core uninstall-dependencies cleanup-mesh-test cleanup-crds cleanup-namespaces
	@echo "$(GREEN)=== Complete cleanup finished$(NC)"
	@echo ""

uninstall-core: uninstall-cloud-core-components uninstall-cloud-core-configuration
	@echo "$(GREEN)=== Core uninstallation completed$(NC)"
	@echo ""

uninstall-cloud-core-components: uninstall-site-management uninstall-config-server uninstall-core-operator uninstall-dbaas-agent uninstall-paas-mediation uninstall-control-plane uninstall-ingress-gateway uninstall-facade-operator
	@echo "$(GREEN)=== Cloud Core components uninstallation completed$(NC)"
	@echo ""

uninstall-dependencies: uninstall-dbaas uninstall-consul uninstall-monitoring uninstall-metrics-server
	@echo "$(GREEN)=== Dependencies uninstallation completed$(NC)"
	@echo ""

# -----------------------------------------Cloud Core components uninstall targets-------------------------------------

uninstall-cloud-core-configuration: uninstall-control-plane-db-credentials
	@echo "$(YELLOW)--- Uninstalling cloud-core-app-chart...$(NC)"
	helm uninstall cloud-core-app-chart -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""
	@echo ""

uninstall-control-plane-db-credentials:
	@echo "$(YELLOW)--- Uninstalling control-plane-db-credentials...$(NC)"
	kubectl delete secret control-plane-db-credentials -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-facade-operator:
	@echo "$(YELLOW)--- Uninstalling facade-operator...$(NC)"
	helm uninstall facade-operator -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-ingress-gateway:
	@echo "$(YELLOW)--- Uninstalling ingress-gateway...$(NC)"
	helm uninstall ingress-gateway -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-site-management:
	@echo "$(YELLOW)--- Uninstalling site-management...$(NC)"
	helm uninstall site-management -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-paas-mediation:
	@echo "$(YELLOW)--- Uninstalling paas-mediation...$(NC)"
	helm uninstall paas-mediation -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-control-plane: uninstall-control-plane-db-credentials
	@echo "$(YELLOW)--- Uninstalling control-plane...$(NC)"
	helm uninstall control-plane -n $(CORE_NAMESPACE) --ignore-not-found=true || true

uninstall-dbaas-agent:
	@echo "$(YELLOW)--- Uninstalling dbaas-agent...$(NC)"
	helm uninstall dbaas-agent -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-core-operator:
	@echo "$(YELLOW)--- Uninstalling core-operator...$(NC)"
	helm uninstall core-operator -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-config-server:
	@echo "$(YELLOW)--- Uninstalling config-server...$(NC)"
	helm uninstall config-server -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

# -----------------------------------------Dependencies uninstall targets-------------------------------------

uninstall-consul:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Uninstalling consul...$(NC)"; \
		helm uninstall consul -n $(CONSUL_NAMESPACE) --ignore-not-found=true || true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Consul uninstallation (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

uninstall-monitoring:
	@if [ "$(INSTALL_MONITORING)" = "true" ]; then \
		echo "$(YELLOW)--- Uninstalling monitoring-operator...$(NC)"; \
		helm uninstall monitoring-operator -n $(MONITORING_NAMESPACE) --ignore-not-found=true || true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Monitoring uninstallation (INSTALL_MONITORING=false)...$(NC)"; \
		echo ""; \
	fi

uninstall-metrics-server:
	@if [ "$(INSTALL_METRICS_SERVER)" = "true" ]; then \
		echo "$(YELLOW)--- Cleaning up metrics-server...$(NC)"; \
		kubectl delete -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml --ignore-not-found=true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping metrics-server cleanup (INSTALL_METRICS_SERVER=false)...$(NC)"; \
		echo ""; \
	fi

cleanup-crds:
	@echo "$(YELLOW)--- Cleaning up CRDs...$(NC)"
	@if [ "$(INSTALL_CRDS)" = "true" ]; then \
		echo "Removing Facade Operator CRDs..."; \
		kubectl delete crd facadeservices.qubership.org --ignore-not-found=true; \
		kubectl delete crd gateways.core.qubership.org --ignore-not-found=true; \
		echo "Removing Core Operator CRDs..."; \
		kubectl delete crd securities.core.qubership.org --ignore-not-found=true; \
		kubectl delete crd meshes.core.qubership.org --ignore-not-found=true; \
		kubectl delete crd dbaases.core.qubership.org --ignore-not-found=true; \
		kubectl delete crd maases.core.qubership.org --ignore-not-found=true; \
		kubectl delete crd composites.core.qubership.org --ignore-not-found=true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping CRD cleanup (INSTALL_CRDS=false)...$(NC)"; \
		echo ""; \
	fi

cleanup-namespaces:
	@echo "$(YELLOW)--- Cleaning up namespaces...$(NC)"
	@if [ "$(CREATE_NAMESPACE)" = "true" ]; then \
		echo "Removing namespaces..."; \
		if [ "$(INSTALL_CONSUL)" = "true" ]; then \
			kubectl delete namespace $(CONSUL_NAMESPACE) --ignore-not-found=true; \
		fi; \
		if [ "$(INSTALL_MONITORING)" = "true" ]; then \
			kubectl delete namespace $(MONITORING_NAMESPACE) --ignore-not-found=true; \
		fi; \
		kubectl delete namespace $(CORE_NAMESPACE) --ignore-not-found=true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping namespace cleanup (CREATE_NAMESPACE=false)...$(NC)"; \
		echo ""; \
	fi

# =============================================================================
# MESH SMOKE TEST
# =============================================================================

mesh-smoke-test: mesh-smoke-test-print-start deploy-mesh-test wait-for-mesh-test-pod run-mesh-test-curl cleanup-mesh-test
	@echo "$(GREEN)=== Cloud Core Mesh Smoke Test completed$(NC)"
	@echo ""

mesh-smoke-test-print-start:
	@echo "$(GREEN)=== Starting Cloud Core Mesh Smoke Test...$(NC)"
	@echo ""

deploy-mesh-test:
	@echo "$(YELLOW)--- Deploying mesh test resources...$(NC)"
	@if [ -f "mesh-test.yaml" ]; then \
		kubectl apply -f mesh-test.yaml -n $(CORE_NAMESPACE); \
		echo "$(GREEN)--- Mesh test resources deployed successfully$(NC)"; \
	else \
		echo "$(RED)Error: mesh-test.yaml not found$(NC)"; \
		echo "$(CYAN)Please create mesh-test.yaml file with your test resources$(NC)"; \
		exit 1; \
	fi
	@echo ""

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

cleanup-mesh-test:
	@echo "$(YELLOW)--- Cleaning up mesh test resources...$(NC)"
	@if [ -f "mesh-test.yaml" ]; then \
		echo "$(CYAN)--- Deleting mesh test resources from $(CORE_NAMESPACE) namespace...$(NC)"; \
		kubectl delete -f mesh-test.yaml -n $(CORE_NAMESPACE) --ignore-not-found=true || true; \
		echo "$(GREEN)--- Mesh test resources cleaned up successfully$(NC)"; \
	fi
	@echo ""

# =============================================================================
# DBAAS
# =============================================================================

deploy-dbaas:
	@if [ "$(INSTALL_DBAAS)" = "true" ]; then \
		echo "$(GREEN)=== Starting DBaaS Installation...$(NC)"; \
		echo ""; \
		if [ -d "dbaas" ]; then \
			echo "$(CYAN)--- Running DBaaS installation from dbaas directory...$(NC)"; \
			$(call run_dbaas_operation,install); \
			echo ""; \
		else \
			echo "$(RED)Error: dbaas directory not found$(NC)"; \
			echo "$(CYAN)Please ensure the dbaas directory exists in the current location$(NC)"; \
			exit 1; \
		fi; \
	else \
		echo "$(CYAN)--- Skipping DBaaS installation (INSTALL_DBAAS=false)...$(NC)"; \
		echo "$(GREEN)=== Running DBaaS smoke test on existing installation...$(NC)"; \
		echo ""; \
		if [ -d "dbaas" ]; then \
			echo "$(CYAN)--- Running DBaaS smoke test from dbaas directory...$(NC)"; \
			$(call run_dbaas_operation,smoke-test); \
		else \
			echo "$(RED)Error: dbaas directory not found$(NC)"; \
			echo "$(CYAN)Please ensure the dbaas directory exists in the current location$(NC)"; \
			exit 1; \
		fi; \
	fi

uninstall-dbaas:
	@if [ "$(INSTALL_DBAAS)" = "true" ]; then \
		echo "$(YELLOW)--- Uninstalling DBaaS components...$(NC)"; \
		if [ -d "dbaas" ]; then \
			echo "$(YELLOW)--- Running DBaaS uninstallation from dbaas directory...$(NC)"; \
			$(call run_dbaas_operation,uninstall); \
			echo ""; \
		else \
			echo "$(RED)Error: dbaas directory not found$(NC)"; \
			echo "$(CYAN)Please ensure the dbaas directory exists in the current location$(NC)"; \
			exit 1; \
		fi; \
	else \
		echo "$(YELLOW)--- Skipping DBaaS uninstallation (INSTALL_DBAAS=false)...$(NC)"; \
		echo ""; \
	fi

# =============================================================================
# MAAS
# =============================================================================

deploy-maas:
	@echo "$(GREEN)=== Starting MaaS installation...$(NC)"
	@echo ""
	@if [ "$(INSTALL_MAAS)" = "true" ]; then \
		$(call run_maas_operation,install); \
	else \
		echo "$(CYAN)--- Skipping MaaS installation (INSTALL_MAAS=false)...$(NC)"; \
		echo ""; \
	fi
	@echo ""

uninstall-maas:
	@echo "$(GREEN)=== Starting MaaS uninstallation...$(NC)"
	@echo ""
	@if [ "$(INSTALL_MAAS)" = "true" ]; then \
		$(call run_maas_operation,uninstall); \
	else \
		echo "$(CYAN)--- Skipping MaaS uninstallation (INSTALL_MAAS=false)...$(NC)"; \
		echo ""; \
	fi
	@echo ""

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

define clone_or_update_repo
	if [ -d "repos/$(1)" ]; then \
		echo "Updating $(1)..."; \
		(cd repos/$(1) && git pull); \
	else \
		echo "Cloning $(1)..."; \
		if [ -n "$(3)" ]; then \
			git clone --depth 1 -b $(3) $(2) repos/$(1); \
		else \
			git clone --depth 1 $(2) repos/$(1); \
		fi; \
	fi
endef

# Helper function to export all variables from .config.mk
define export_config_vars
	bash -c "$$(tr -d '\r' < $(CONFIG_FILE) | sed 's/^\([A-Z_][A-Z0-9_]*\) ?= \(.*\)/export \1=\"\2\"/' && \
	echo 'envsubst < ./core-values.envsubst > ./core-values.yaml')"
endef

# Helper function to check namespace existence
define check_namespace_existence
	set -e; \
	echo "$(CYAN)--- Checking if namespace $(1) exists...$(NC)"; \
	kubectl get namespace $(1) >/dev/null 2>&1 || (echo "$(RED)Error: Namespace $(1) not found$(NC)" && echo "$(CYAN)Please create the namespace first:$(NC)" && echo "$(CYAN)kubectl create namespace $(1)$(NC)" && exit 1); \
	echo "$(GREEN)--- Namespace $(1) found$(NC)"
endef

# Helper function to check and create namespace if it doesn't exist
define ensure_namespace_exists
	echo "$(CYAN)--- Checking if namespace $(1) exists...$(NC)"; \
	if kubectl get namespace $(1) >/dev/null 2>&1; then \
		echo "$(GREEN)--- Namespace $(1) already exists$(NC)"; \
	else \
		echo "$(YELLOW)--- Namespace $(1) not found, creating it...$(NC)"; \
		kubectl create namespace $(1); \
		echo "$(GREEN)--- Namespace $(1) created successfully$(NC)"; \
	fi
endef

# Helper function to check CRD existence
define check_crd_existence
	set -e; \
	echo "$(CYAN)--- Checking if CRD $(1) exists...$(NC)"; \
	kubectl api-resources | grep -q $(2) || (echo "$(RED)Error: CRD $(1) not found$(NC)" && echo "$(CYAN)Please install the CRD first:$(NC)" && echo "$(CYAN)$(3)$(NC)" && exit 1); \
	echo "$(GREEN)--- CRD $(1) found$(NC)"
endef

# Helper function to update Helm dependencies
define update_helm_dependencies
	if [ -n "$(2)" ]; then \
		echo "$(CYAN)--- Updating Helm dependencies in $(1) with config $(2)...$(NC)"; \
		if (cd $(1) && helm dependency update --repository-config $(2)); then \
			echo "$(GREEN)--- Helm dependencies updated successfully$(NC)"; \
		else \
			echo "$(RED)Error: Failed to update Helm dependencies$(NC)"; \
			exit 1; \
		fi; \
	else \
		echo "$(CYAN)--- Updating Helm dependencies in $(1) with config ./core-bootstrap-helm-repo.yaml...$(NC)"; \
		if (cd $(1) && helm dependency update --repository-config ./core-bootstrap-helm-repo.yaml); then \
			echo "$(GREEN)--- Helm dependencies updated successfully$(NC)"; \
		else \
			echo "$(RED)Error: Failed to update Helm dependencies$(NC)"; \
			exit 1; \
		fi; \
	fi
endef

# Helper function to run DBaaS operations
define run_dbaas_operation
	set -e; \
	echo "$(CYAN)--- Using config file: $(DBAAS_CONFIG_FILE)$(NC)"; \
	echo "$(CYAN)--- Using dbaas service name: $(DBAAS_SERVICE_NAME)$(NC)"; \
	echo "$(CYAN)--- Using dbaas namespace: $(DBAAS_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using pg namespace: $(PG_NAMESPACE)$(NC)"; \
	SKIP_CRDS=$(if $(filter true,$(INSTALL_CRDS)),false,true); \
	echo "$(CYAN)--- Using skip crds: $$SKIP_CRDS$(NC)"; \
	if (cd dbaas && '$(MAKE)' $(1) CONFIG_FILE=$(DBAAS_CONFIG_FILE) DBAAS_SERVICE_NAME=$(DBAAS_SERVICE_NAME) DBAAS_NAMESPACE=$(DBAAS_NAMESPACE) PG_NAMESPACE=$(PG_NAMESPACE) SKIP_CRDS=$$SKIP_CRDS CREATE_NAMESPACE=$(CREATE_NAMESPACE)); then \
		echo "$(GREEN)=== DBaaS $(1) completed successfully$(NC)"; \
	else \
		echo "$(RED)Error: DBaaS $(1) failed$(NC)"; \
		exit 1; \
	fi
endef

# Helper function to run MAAS operations
define run_maas_operation
	set -e; \
	echo "$(CYAN)--- Using config file: $(MAAS_CONFIG_FILE)$(NC)"; \
	echo "$(CYAN)--- Using MAAS namespace: $(MAAS_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using RabbitMQ namespace: $(RABBIT_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using Kafka namespace: $(KAFKA_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using Kafka instances: $(KAFKA_INSTANCES)$(NC)"; \
	echo "$(CYAN)--- Using RabbitMQ instances: $(RABBIT_INSTANCES)$(NC)"; \
	echo "$(CYAN)--- Using DBAAS namespace: $(DBAAS_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using DBAAS service name: $(DBAAS_SERVICE_NAME)$(NC)"; \
	if (cd maas && '$(MAKE)' $(1) CONFIG_FILE=$(MAAS_CONFIG_FILE) MAAS_NAMESPACE=$(MAAS_NAMESPACE) RABBIT_NAMESPACE=$(RABBIT_NAMESPACE) KAFKA_NAMESPACE=$(KAFKA_NAMESPACE) KAFKA_INSTANCES=$(KAFKA_INSTANCES) RABBIT_INSTANCES=$(RABBIT_INSTANCES) CREATE_NAMESPACE=$(CREATE_NAMESPACE) DBAAS_NAMESPACE=$(DBAAS_NAMESPACE) DBAAS_SERVICE_NAME=$(DBAAS_SERVICE_NAME)); then \
		echo "$(GREEN)=== MAAS $(1) completed successfully$(NC)"; \
	else \
		echo "$(RED)Error: MAAS $(1) failed$(NC)"; \
		exit 1; \
	fi
endef

