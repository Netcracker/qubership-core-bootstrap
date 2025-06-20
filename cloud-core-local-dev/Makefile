.PHONY: all minikube-setup clone-repos deploy-paas-mediation clean deploy-control-plane \
	deploy-ingress-gateway add-helm-repos do-magic deploy-consul deploy-monitoring deploy-cloud-core \
	deploy-cloud-core-print-start deploy-paas-mediation \
	deploy-control-plane apply-facade-operator-crds deploy-facade-operator deploy-site-management \
	deploy-ingress-gateway prepare-core-bootstrap prepare-core-bootstrap-print-start \
	deploy-core-bootstrap-image create-finalyzer-service-account \
	build-and-load-cr-synchronizer-image deploy-patroni-core deploy-patroni-core-print-start clone-patroni-core-repos \
	update-patroni-storage-class apply-patroni-core-crds install-patroni-core wait-for-patroni-pods \
	deploy-dbaas deploy-dbaas-print-start clone-dbaas-repos label-database-node \
	install-dbaas deploy-patroni-services deploy-patroni-services-print-start \
	install-patroni-services wait-for-patroni-services deploy-dbaas-with-infra

# Default target
all: do-magic

ifeq ($(DEBUG),true)
  HELM_DEBUG_FLAGS := --debug --dry-run
else
  HELM_DEBUG_FLAGS :=
endif

# Profile definition
PROFILE ?= cloud-core-dev

# Color definitions
YELLOW := \033[1;33m
GREEN := \033[1;32m
NC := \033[0m


do-magic: minikube-setup clone-repos deploy-metrics-server deploy-monitoring deploy-consul deploy-dbaas-with-infra prepare-core-bootstrap deploy-cloud-core

#---------------------------- minikube ---------------------------------------------

minikube-setup: minikube-setup-print-start create-minikube-profile label-database-node
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

label-database-node:
	@echo "$(YELLOW)--- Checking for existing database node...$(NC)"
	@if ! kubectl get nodes --selector=region=database | grep -q "$(PROFILE)"; then \
		echo "Labeling $(PROFILE) node with region=database..."; \
		kubectl label nodes $(PROFILE) region=database --overwrite; \
	else \
		echo "Node $(PROFILE) already has region=database label"; \
	fi
	@echo ""

minikube-delete-profile:
	@echo "$(YELLOW)--- Cleaning up resources...$(NC)"
	minikube delete -p=$(PROFILE)
	@echo ""

# ----------------------------------------------------------------------------

clone-repos:
	@echo "$(YELLOW)--- Cloning required repositories...$(NC)"
	mkdir -p repos
	$(call clone_or_update_repo,qubership-monitoring-operator,https://github.com/Netcracker/qubership-monitoring-operator.git,main)
	$(call clone_or_update_repo,qubership-core-paas-mediation,https://github.com/Netcracker/qubership-core-paas-mediation.git,feature/local-deployment)
	$(call clone_or_update_repo,qubership-core-control-plane,https://github.com/Netcracker/qubership-core-control-plane.git,feature/local-deployment)
	$(call clone_or_update_repo,qubership-core-ingress-gateway,https://github.com/Netcracker/qubership-core-ingress-gateway.git,feature/local-deployment)
	$(call clone_or_update_repo,qubership-core-site-management,https://github.com/Netcracker/qubership-core-site-management.git,feature/local-deployment)
	$(call clone_or_update_repo,qubership-core-facade-operator,https://github.com/Netcracker/qubership-core-facade-operator.git,feature/local-deployment)
	$(call clone_or_update_repo,qubership-core-dbaas-agent,https://github.com/Netcracker/qubership-core-dbaas-agent.git,feature/local-deployment)
	$(call clone_or_update_repo,qubership-core-core-operator,https://github.com/Netcracker/qubership-core-core-operator.git,feature/local-deployment)
	$(call clone_or_update_repo,qubership-core-config-server,https://github.com/Netcracker/qubership-core-config-server.git,feature/local-deployment)
	@echo ""

# -------------------------- consul -----------------------

deploy-consul-old:
	@echo "$(GREEN)=== Deploying Consul...$(NC)"
	@echo ""
	helm upgrade --install --values ./consul-chart-3/values.yaml consul ./consul-chart-3/ --namespace consul --create-namespace
	@echo "$(GREEN)=== Consul is deployed$(NC)"
	@echo ""

deploy-consul: deploy-consul-print-start clone-consul-repo change-consul-values install-consul
	@echo "$(GREEN)=== Consul setup completed...$(NC)"
	@echo ""

deploy-consul-print-start:
	@echo "$(GREEN)=== Starting Consul setup...$(NC)"
	@echo ""

clone-consul-repo:
	@echo "$(YELLOW)--- Cloning consul repository...$(NC)"
	$(call clone_or_update_repo,consul-k8s,https://github.com/hashicorp/consul-k8s,main)

change-consul-values:
	@echo "$(YELLOW)--- Updating Consul Helm chart security context...$(NC)"
	@if [ -f ./repos/consul-k8s/charts/consul/templates/_helpers.tpl ]; then \
		sed -i 's/runAsNonRoot: true/runAsNonRoot: false/g' ./repos/consul-k8s/charts/consul/templates/_helpers.tpl; \
		sed -i 's/runAsUser: 100/runAsUser: 0/g' ./repos/consul-k8s/charts/consul/templates/_helpers.tpl; \
		echo "Security context updated in _helpers.tpl"; \
	else \
		echo "Error: _helpers.tpl not found"; \
		exit 1; \
	fi
	@echo ""

install-consul:
	@echo "$(YELLOW)--- Installing consul...$(NC)"
	helm upgrade --install --namespace consul --create-namespace \
 		consul ./repos/consul-k8s/charts/consul/ \
 		-f ./repos/consul-k8s/charts/consul/values.yaml \
 		--set server.securityContext.runAsNonRoot=false \
 		--set server.securityContext.runAsUser=0

# todo - add waiting step

# ----------------- monitoring --------------------------

deploy-monitoring:
	@echo "$(GREEN)=== Deploying Monitoring...$(NC)"
	@echo ""
	helm upgrade --install --namespace monitoring --create-namespace \
		monitoring-operator ./repos/qubership-monitoring-operator/charts/qubership-monitoring-operator \
		-f ./repos/qubership-monitoring-operator/charts/qubership-monitoring-operator/values.yaml
	@echo "$(GREEN)=== Monitoring is deployed$(NC)"
	@echo ""

# ------------------- metrics-server -------------------------------

deploy-metrics-server:
	@echo "$(GREEN)=== Deploying Metrics Server...$(NC)"
	@echo ""
	kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
	kubectl patch deployment metrics-server -n kube-system --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]'
	@echo "$(GREEN)=== Metrics Server is deployed$(NC)"
	@echo ""

# ----------- patroni-core ------------------------------

deploy-patroni-core: deploy-patroni-core-print-start clone-patroni-core-repos update-patroni-storage-class apply-patroni-core-crds install-patroni-core wait-for-patroni-pods
	@echo "$(GREEN)=== Patroni core installation completed$(NC)"

deploy-patroni-core-print-start:
	@echo "$(GREEN)=== Starting patroni core installation...$(NC)"
	@echo ""

clone-patroni-core-repos:
	@echo "$(YELLOW)--- Cloning required repositories...$(NC)"
	mkdir -p repos
	git clone https://github.com/Netcracker/pgskipper-operator.git repos/pgskipper-operator || echo "Directory already exists"
	@echo ""

update-patroni-storage-class: clone-patroni-core-repos
	@echo "$(YELLOW)--- Updating storage class in patroni quickstart sample files...$(NC)"
	@if [ -f ./repos/pgskipper-operator/charts/patroni-core/patroni-core-quickstart-sample.yaml ]; then \
		sed -i 's/storageClass:.*/storageClass: "standard"/' ./repos/pgskipper-operator/charts/patroni-core/patroni-core-quickstart-sample.yaml; \
		echo "Storage class updated in patroni-core-quickstart-sample.yaml"; \
	else \
		echo "Error: patroni-core-quickstart-sample.yaml not found"; \
		exit 1; \
	fi
	@if [ -f ./repos/pgskipper-operator/charts/patroni-services/patroni-services-quickstart-sample.yaml ]; then \
		sed -i 's/storageClass:.*/storageClass: "standard"/' ./repos/pgskipper-operator/charts/patroni-services/patroni-services-quickstart-sample.yaml; \
		echo "Storage class updated in patroni-services-quickstart-sample.yaml"; \
	else \
		echo "Error: patroni-services-quickstart-sample.yaml not found"; \
		exit 1; \
	fi
	@echo "Storage class update completed"
	@echo ""

apply-patroni-core-crds: clone-patroni-core-repos
	@echo "$(YELLOW)--- Create CRD for pg-operator$(NC)"
	@if ! kubectl get crd patronicores.qubership.org >/dev/null 2>&1; then \
		echo "Creating patronicores.qubership.org CRD..."; \
		kubectl create -f ./repos/pgskipper-operator/charts/patroni-core/crds/qubership.org_patronicores.yaml; \
	else \
		echo "patronicores.qubership.org CRD already exists"; \
	fi
	@if ! kubectl get crd patroniservices.qubership.org >/dev/null 2>&1; then \
		echo "Creating patroniservices.qubership.org CRD..."; \
		kubectl create -f ./repos/pgskipper-operator/charts/patroni-services/crds/qubership.org_patroniservices.yaml; \
	else \
		echo "patroniservices.qubership.org CRD already exists"; \
	fi
	@echo ""

install-patroni-core: clone-patroni-core-repos update-patroni-storage-class apply-patroni-core-crds
	@echo "$(YELLOW)--- Deploying PG operator...$(NC)"
	helm upgrade --install --namespace=postgres --create-namespace \
 		 patroni-core ./repos/pgskipper-operator/charts/patroni-core \
 		 -f ./repos/pgskipper-operator/charts/patroni-core/patroni-core-quickstart-sample.yaml \
 		 --set postgresPassword="password"
	@echo ""

wait-for-patroni-pods: install-patroni-core
	@echo "$(YELLOW)--- Waiting for patroni pods to be ready (timeout: 5 minutes)...$(NC)"
	@TIMEOUT=600; \
	START_TIME=$$(date +%s); \
	while true; do \
		CURRENT_TIME=$$(date +%s); \
		ELAPSED_TIME=$$((CURRENT_TIME - START_TIME)); \
		if [ $$ELAPSED_TIME -ge $$TIMEOUT ]; then \
			echo "Timeout reached after 10 minutes. Not all pods are running."; \
			exit 1; \
		fi; \
		RUNNING_PATRONI_PODS=$$(kubectl -n postgres get pods --selector=app=patroni --field-selector=status.phase=Running -o name | wc -l); \
		RUNNING_MASTER_PODS=$$(kubectl -n postgres get pods --selector=pgtype=master --field-selector=status.phase=Running -o name | wc -l); \
		if [ "$$RUNNING_PATRONI_PODS" -eq 2 ] && [ "$$RUNNING_MASTER_PODS" -eq 1 ]; then \
			echo "All patroni pods are running ($$RUNNING_PATRONI_PODS/2) and master pod is ready ($$RUNNING_MASTER_PODS/1)!"; \
			break; \
		fi; \
		echo "Waiting for pods to be ready... (Patroni: $$RUNNING_PATRONI_PODS/2, Master: $$RUNNING_MASTER_PODS/1) - $$((TIMEOUT - ELAPSED_TIME))s remaining"; \
		sleep 10; \
	done
	@echo ""

# --------------------- dbaas ----------------------------

deploy-dbaas: deploy-dbaas-print-start clone-dbaas-repos label-database-node install-dbaas
	@echo "$(GREEN)=== DBaaS installation completed$(NC)"
	@echo ""

deploy-dbaas-print-start:
	@echo "$(GREEN)=== Starting DBaaS installation...$(NC)"
	@echo ""

clone-dbaas-repos:
	@echo "$(YELLOW)--- Cloning required repositories...$(NC)"
	mkdir -p repos
	git clone -b main https://github.com/Netcracker/qubership-dbaas.git repos/qubership-dbaas || echo "Directory already exists"
	@echo ""

install-dbaas:
	@echo "$(YELLOW)--- Deploying DBaaS...$(NC)"
	helm upgrade --install -n dbaas --create-namespace \
		dbaas-aggregator ./repos/qubership-dbaas/helm-templates/dbaas-aggregator \
		-f ./repos/qubership-dbaas/helm-templates/dbaas-aggregator/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./dbaas-values.yaml \
		--set SERVICE_NAME=dbaas-aggregator \
		--set IMAGE_REPOSITORY=ghcr.io/netcracker/qubership-dbaas \
		--set TAG=latest \
		$(HELM_DEBUG_FLAGS)
	@echo ""

# --------------------- patroni-services -------------------------------

deploy-patroni-services: deploy-patroni-services-print-start clone-patroni-core-repos update-patroni-storage-class apply-patroni-core-crds install-patroni-services wait-for-patroni-services
	@echo "$(GREEN)=== Patroni services installation completed$(NC)"
	@echo ""

deploy-patroni-services-print-start:
	@echo "$(GREEN)=== Starting patroni services installation...$(NC)"
	@echo ""

install-patroni-services:
	@echo "$(YELLOW)--- Deploying patroni-services...$(NC)"
	helm upgrade --install --namespace=postgres --create-namespace \
 		-f ./repos/pgskipper-operator/charts/patroni-services/patroni-services-quickstart-sample.yaml \
 		 patroni-services ./repos/pgskipper-operator/charts/patroni-services \
 		 --set dbaas.install=true \
 		 --set dbaas.aggregator.registrationAddress=http://dbaas-aggregator.dbaas.svc.cluster.local:8080 \
 		 --set dbaas.aggregator.registrationUsername=cluster-dba \
 		 --set dbaas.aggregator.registrationPassword=password \
 		 --set postgresPassword="password"
	@echo ""

wait-for-patroni-services: install-patroni-services
	@echo "$(YELLOW)--- Waiting for patroni services registration (timeout: 5 minutes)...$(NC)"
	@TIMEOUT=300; \
	START_TIME=$$(date +%s); \
	while true; do \
		CURRENT_TIME=$$(date +%s); \
		ELAPSED_TIME=$$((CURRENT_TIME - START_TIME)); \
		if [ $$ELAPSED_TIME -ge $$TIMEOUT ]; then \
			echo "Timeout reached after 5 minutes. Registration message not found in logs."; \
			exit 1; \
		fi; \
		if kubectl logs -n postgres -l app=dbaas-postgres-adapter --tail=100 | grep -q "Registration finished"; then \
			echo "$(GREEN)=== Patroni services registration completed$(NC)"; \
			break; \
		fi; \
		echo "Waiting for patroni services registration... - $$((TIMEOUT - ELAPSED_TIME))s remaining"; \
		sleep 10; \
	done
	@echo ""

# ---------------------- FULL DBaaS ---------------------------------------

deploy-dbaas-with-infra: deploy-patroni-core deploy-dbaas deploy-patroni-services

# --------------------- bootstrap --------------------

prepare-core-bootstrap: prepare-core-bootstrap-print-start deploy-cloud-core-configuration
	@echo "$(GREEN)=== Bootstrap preparation completed$(NC)"
	@echo ""

prepare-core-bootstrap-print-start:
	@echo "$(GREEN)=== Starting bootstrap preparation...$(NC)"
	@echo ""

deploy-cloud-core-configuration:
	@echo "$(YELLOW)--- Deploying cloud-core-configuration$(NC)"
	helm upgrade --install -n cloud-core --create-namespace \
		cloud-core-app-chart ../cloud-core-configuration/helm-charts \
		-f ../cloud-core-configuration/helm-charts/values.yaml \
		-f ./general-cluster-values.yaml \
		--set CONSUL_ENABLED=false \
		--set NAMESPACE=cloud-core \
		--set CORE_BOOTSTRAP_IMAGE=ghcr.io/netcracker/core-bootstrap:latest \
		$(HELM_DEBUG_FLAGS)
	@echo ""

# ------------------ cloud core ---------------------------

deploy-cloud-core: add-helm-repos deploy-facade-operator deploy-ingress-gateway deploy-control-plane deploy-paas-mediation deploy-dbaas-agent deploy-core-operator deploy-config-server deploy-site-management
	@echo "$(GREEN)=== Cloud Core installation completed$(NC)"
	@echo ""

deploy-cloud-core-print-start:
	@echo "$(GREEN)=== Starting Cloud Core installation...$(NC)"
	@echo ""

add-helm-repos:
	@echo "$(YELLOW)--- Adding required helm repositories...$(NC)"
	helm repo add coretpl https://netcracker.github.io/qubership-core-bootstrap/
	helm repo add hashicorp https://helm.releases.hashicorp.com
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	@echo ""

apply-facade-operator-crds:
	@echo "$(YELLOW)--- Create Facade Operator CRDs...$(NC)"
	@if ! kubectl get crd facadeservices.qubership.org >/dev/null 2>&1; then \
		echo "Creating facadeservices.qubership.org CRD..."; \
		kubectl apply -f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/new-crd/facade-service-crd-v1.yaml; \
	else \
		echo "facadeservices.qubership.org CRD already exists"; \
	fi
	@if ! kubectl get crd gateways.core.qubership.org >/dev/null 2>&1; then \
		echo "Creating gateways.core.qubership.org CRD..."; \
		kubectl apply -f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/new-crd/gateway-crd-v1.yaml; \
	else \
		echo "gateways.core.qubership.org CRD already exists"; \
	fi
	@echo ""

deploy-facade-operator: clone-repos prepare-core-bootstrap apply-facade-operator-crds
	@echo "$(YELLOW)--- Setting up Facade Operator...$(NC)"
	(cd repos/qubership-core-facade-operator/helm-templates/facade-operator && helm dependency update)
	helm upgrade --install facade-operator ./repos/qubership-core-facade-operator/helm-templates/facade-operator -n cloud-core \
		-f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/values.yaml \
		-f ./repos/qubership-core-facade-operator/helm-templates/facade-operator/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./general-core-values.yaml \
		--set TAG=latest \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-ingress-gateway: clone-repos prepare-core-bootstrap
	@echo "$(YELLOW)--- Setting up Ingress Gateway...$(NC)"
	(cd repos/qubership-core-ingress-gateway/helm-templates/frontend-gateway && helm dependency update)
	helm upgrade --install ingress-gateway ./repos/qubership-core-ingress-gateway/helm-templates/frontend-gateway -n cloud-core \
		-f ./repos/qubership-core-ingress-gateway/helm-templates/frontend-gateway/values.yaml \
		-f ./repos/qubership-core-ingress-gateway/helm-templates/frontend-gateway/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./general-core-values.yaml \
		--set CLOUD_PUBLIC_HOST=svc.cluster.local \
		--set CLOUD_PRIVATE_HOST=svc.cluster.local \
		--set TAG=main-20250325151826-6 \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-site-management: clone-repos prepare-core-bootstrap
	@echo "$(YELLOW)--- Setting up Site Management...$(NC)"
	(cd repos/qubership-core-site-management/helm-templates/site-management && helm dependency update)
	helm upgrade --install site-management ./repos/qubership-core-site-management/helm-templates/site-management -n cloud-core \
		-f ./repos/qubership-core-site-management/helm-templates/site-management/values.yaml \
		-f ./repos/qubership-core-site-management/helm-templates/site-management/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./general-core-values.yaml \
		--set TAG=feature-local-deployment-snapshot \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-paas-mediation: clone-repos prepare-core-bootstrap
	@echo "$(YELLOW)--- Setting up PaaS Mediation...$(NC)"
	(cd repos/qubership-core-paas-mediation/helm-templates/paas-mediation && helm dependency update)
	helm upgrade --install -n cloud-core --create-namespace \
		paas-mediation ./repos/qubership-core-paas-mediation/helm-templates/paas-mediation \
		-f ./repos/qubership-core-paas-mediation/helm-templates/paas-mediation/values.yaml \
		-f ./repos/qubership-core-paas-mediation/helm-templates/paas-mediation/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./general-core-values.yaml \
		--set TAG=latest \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-control-plane: clone-repos prepare-core-bootstrap
	@echo "$(YELLOW)--- Setting up Control Plane...$(NC)"
	(cd repos/qubership-core-control-plane/helm-templates/control-plane && helm dependency update)
	helm upgrade --install -n cloud-core --create-namespace \
		control-plane ./repos/qubership-core-control-plane/helm-templates/control-plane \
		-f ./repos/qubership-core-control-plane/helm-templates/control-plane/values.yaml \
		-f ./repos/qubership-core-control-plane/helm-templates/control-plane/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./general-core-values.yaml \
		--set TAG=fix-dockerfile-snapshot \
		$(HELM_DEBUG_FLAGS)
	@echo ""


deploy-dbaas-agent: clone-repos prepare-core-bootstrap
	@echo "$(YELLOW)--- Setting up DBaaS agent...$(NC)"
	(cd repos/qubership-core-dbaas-agent/helm-templates/dbaas-agent && helm dependency update)
	helm upgrade --install dbaas-agent ./repos/qubership-core-dbaas-agent/helm-templates/dbaas-agent -n cloud-core \
		-f ./repos/qubership-core-dbaas-agent/helm-templates/dbaas-agent/values.yaml \
		-f ./repos/qubership-core-dbaas-agent/helm-templates/dbaas-agent/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./general-core-values.yaml \
		--set TAG=latest \
		$(HELM_DEBUG_FLAGS)
	@echo ""

apply-core-operator-crds:
	@echo "$(YELLOW)--- Applying Core Operator CRDs...$(NC)"
	kubectl apply -f ./repos/qubership-core-core-operator/helm-templates/core-operator/new-crd
	@echo ""

deploy-core-operator: clone-repos prepare-core-bootstrap apply-core-operator-crds
	@echo "$(YELLOW)--- Setting up Core Operator...$(NC)"
	(cd repos/qubership-core-core-operator/helm-templates/core-operator && helm dependency update)
	helm upgrade --install core-operator ./repos/qubership-core-core-operator/helm-templates/core-operator -n cloud-core \
		-f ./repos/qubership-core-core-operator/helm-templates/core-operator/values.yaml \
		-f ./repos/qubership-core-core-operator/helm-templates/core-operator/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./general-core-values.yaml \
		--set IMAGE_REPOSITORY=core-operator \
		--set TAG=latest \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-config-server: clone-repos prepare-core-bootstrap
	@echo "$(YELLOW)--- Setting up Config Server...$(NC)"
	(cd repos/qubership-core-config-server/helm-templates/config-server && helm dependency update)
	helm upgrade --install config-server ./repos/qubership-core-config-server/helm-templates/config-server -n cloud-core \
		-f ./repos/qubership-core-config-server/helm-templates/config-server/values.yaml \
		-f ./repos/qubership-core-config-server/helm-templates/config-server/resource-profiles/dev.yaml \
		-f ./general-cluster-values.yaml \
		-f ./general-core-values.yaml \
		--set IMAGE_REPOSITORY=config-server \
		--set TAG=latest \
		--set CONSUL_ENABLED=false \
		$(HELM_DEBUG_FLAGS)
	@echo ""

# ------------------------------------------------------------------------------

define clone_or_update_repo
	@if [ -d "repos/$(1)" ]; then \
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