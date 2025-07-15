# MaaS Installation Makefile

# Include configuration
CONFIG_FILE ?= local.mk
include $(CONFIG_FILE)

# Define colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
BLUE := \033[0;34m
RED := \033[0;31m
NC := \033[0m

# Conditional helm options
CREATE_NAMESPACE_OPTION := $(if $(filter true,$(CREATE_NAMESPACE)),--create-namespace,)

# Default target
.PHONY: help
help:
	@echo "$(BLUE)=== MaaS Installation Makefile ===$(NC)"
	@echo ""
	@echo "Usage: make <target> [CONFIG_FILE=local.mk]"
	@echo ""
	@echo "Targets:"
	@echo "  install     - Install all MaaS components"
	@echo "  uninstall   - Uninstall all MaaS components"
	@echo "  validate    - Validate configuration and prerequisites"
	@echo "  show-config - Show current configuration"
	@echo "  clean       - Clean up repositories"
	@echo "  help        - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make install CONFIG_FILE=local.mk"
	@echo "  make uninstall CONFIG_FILE=local.mk"
	@echo "  make validate CONFIG_FILE=local.mk"

# Show configuration
.PHONY: show-config
show-config:
	@echo "$(BLUE)=== ENVIRONMENT VARIABLES ===$(NC)"
	@echo "$(CYAN)--- MaaS values file: $(NC)$(MAAS_VALUES_FILE)"
	@echo "$(CYAN)--- MaaS profile file: $(NC)$(MAAS_PROFILE_FILE)"
	@echo "$(CYAN)--- MaaS namespace: $(NC)$(MAAS_NAMESPACE)"
	@echo "$(CYAN)--- RabbitMQ namespace: $(NC)$(RABBIT_NAMESPACE)"
	@echo "$(CYAN)--- Kafka namespace: $(NC)$(KAFKA_NAMESPACE)"
	@echo "$(CYAN)--- DBaaS namespace: $(NC)$(DBAAS_NAMESPACE)"
	@echo "$(CYAN)--- DBaaS aggregator address: $(NC)$(DBAAS_AGGREGATOR_ADDRESS)"
	@echo "$(CYAN)--- Image tag: $(NC)$(TAG)"
	@echo ""

# Validate prerequisites
.PHONY: validate
validate:
	@echo "$(BLUE)=== PREREQUISITE CHECKS ===$(NC)"
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
	@echo "$(BLUE)=== NAMESPACE CHECK ===$(NC)"
ifneq ($(CREATE_NAMESPACE),true)
	@echo "$(CYAN)--- Checking if required namespaces exist...$(NC)"
	@kubectl get namespace $(MAAS_NAMESPACE) >/dev/null 2>&1 || (echo "$(RED)Error: Namespace $(MAAS_NAMESPACE) not found$(NC)" && echo "$(CYAN)Please create the namespace first:$(NC)" && echo "$(CYAN)kubectl create namespace $(MAAS_NAMESPACE)$(NC)" && exit 1)
	@echo "$(GREEN)--- Namespace $(MAAS_NAMESPACE) found$(NC)"
	@kubectl get namespace $(RABBIT_NAMESPACE) >/dev/null 2>&1 || (echo "$(RED)Error: Namespace $(RABBIT_NAMESPACE) not found$(NC)" && echo "$(CYAN)Please create the namespace first:$(NC)" && echo "$(CYAN)kubectl create namespace $(RABBIT_NAMESPACE)$(NC)" && exit 1)
	@echo "$(GREEN)--- Namespace $(RABBIT_NAMESPACE) found$(NC)"
	@kubectl get namespace $(KAFKA_NAMESPACE) >/dev/null 2>&1 || (echo "$(RED)Error: Namespace $(KAFKA_NAMESPACE) not found$(NC)" && echo "$(CYAN)Please create the namespace first:$(NC)" && echo "$(CYAN)kubectl create namespace $(KAFKA_NAMESPACE)$(NC)" && exit 1)
	@echo "$(GREEN)--- Namespace $(KAFKA_NAMESPACE) found$(NC)"
else
	@echo "$(CYAN)--- Skipping namespace check (CREATE_NAMESPACE=true)...$(NC)"
endif
	@echo ""
	@echo "$(BLUE)=== DBAAS CHECK ===$(NC)"
	@NUMBER_OF_DBAAS_PODS=$$(kubectl -n $(DBAAS_NAMESPACE) get pods --selector=name=dbaas-aggregator --field-selector=status.phase=Running --no-headers | wc -l); \
	if [ "$$NUMBER_OF_DBAAS_PODS" -eq 1 ]; then \
		echo "$(GREEN)--- DBaaS is ready$(NC)"; \
	else \
		echo "$(RED)--- Error: DBaaS is not ready!$(NC)"; \
		echo "$(CYAN)--- Please install the DBaaS first$(NC)"; \
		exit 1; \
	fi
	@echo ""

# Setup repositories
.PHONY: setup-repos
setup-repos:
	@echo "$(BLUE)=== STAGE 1: Repository Setup ===$(NC)"
	@echo "$(CYAN)--- Stage 1 BEGIN: Cloning repositories...$(NC)"
	@mkdir -p repos
	@git clone -b feature/local-deployment --depth 1 https://github.com/Netcracker/qubership-maas.git repos/qubership-maas 2>/dev/null || echo "Directory already exists"
	@echo "$(GREEN)--- Stage 1 END: Repositories ready$(NC)"
	@echo ""

# Install MaaS Service
.PHONY: install-maas-service
install-maas-service:
	@echo "$(BLUE)=== STAGE 2: MaaS Service Installation ===$(NC)"
	@echo "$(CYAN)--- Stage 2 BEGIN: Installing MaaS Service...$(NC)"
	@envsubst < $(MAAS_VALUES_FILE) | helm upgrade --install -n $(MAAS_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		maas-service ./repos/qubership-maas/helm-templates/maas-service \
		-f - \
		-f $(MAAS_PROFILE_FILE) || \
		(echo "$(RED)Error: MaaS Service installation failed$(NC)" && echo "$(CYAN)Please check the helm output above for details$(NC)" && exit 1)
	@echo "$(GREEN)--- Stage 2 END: MaaS Service installation completed$(NC)"
	@echo ""

# Wait for MaaS service started
.PHONY: wait-for-maas-service-started
wait-for-maas-service-started:
	@echo "$(BLUE)=== STAGE 3: Wait for MaaS service started  ===$(NC)"
	@echo "$(CYAN)--- Stage 3 BEGIN: Waiting for MaaS service started (timeout: 5 minutes)...$(NC)"
	@TIMEOUT=300; START_TIME=$$(date +%s); \
	while true; do \
		CURRENT_TIME=$$(date +%s); \
		ELAPSED_TIME=$$((CURRENT_TIME - START_TIME)); \
		if [ $$ELAPSED_TIME -ge $$TIMEOUT ]; then \
			echo "Timeout reached after 5 minutes. Started message not found in logs."; \
			exit 1; \
		fi; \
		MAAS_STARTED=$$(kubectl logs -n $(MAAS_NAMESPACE) -l name=maas-service --tail=100 | grep -q "Starting server on" && echo true || echo false); \
		if [ "$$MAAS_STARTED" = "true" ]; then \
			echo "$(GREEN)=== MaaS service started$(NC)"; \
			break; \
		fi; \
		echo "Waiting for MaaS service start... - $$((TIMEOUT - ELAPSED_TIME))s remaining"; \
		sleep 10; \
	done
	@echo "$(GREEN)--- Stage 3 END: MaaS service is ready$(NC)"
	@echo ""

# Install RabbitMQ
.PHONY: install-rabbitmq
install-rabbitmq:
	@echo "$(BLUE)=== STAGE 4: RabbitMQ Installation ===$(NC)"
	@echo "$(CYAN)--- Stage 4 BEGIN: Installing RabbitMQ...$(NC)"
ifeq ($(CREATE_NAMESPACE),true)
	@kubectl get namespace $(RABBIT_NAMESPACE) || kubectl create namespace $(RABBIT_NAMESPACE)
endif
	@kubectl apply -f charts/rabbit.yaml -n $(RABBIT_NAMESPACE) || \
		(echo "$(RED)Error: RabbitMQ installation failed$(NC)" && echo "$(CYAN)Please check the helm output above for details$(NC)" && exit 1)
	@echo "$(GREEN)--- Stage 4 END: RabbitMQ installation completed$(NC)"
	@echo ""

# Wait for Rabbit server started
.PHONY: wait-for-rabbit-server-started
wait-for-rabbit-server-started:
	@echo "$(BLUE)=== STAGE 5: Wait for RabbitMQ server started  ===$(NC)"
	@echo "$(CYAN)--- Stage 5 BEGIN: Waiting for rabbit server started (timeout: 5 minutes)...$(NC)"
	@TIMEOUT=300; START_TIME=$$(date +%s); \
	while true; do \
		CURRENT_TIME=$$(date +%s); \
		ELAPSED_TIME=$$((CURRENT_TIME - START_TIME)); \
		if [ $$ELAPSED_TIME -ge $$TIMEOUT ]; then \
			echo "Timeout reached after 5 minutes. Started message not found in logs."; \
			exit 1; \
		fi; \
		RABBIT1_STARTED=$$(kubectl logs -n $(RABBIT_NAMESPACE) -l app=rabbitmq-1 --tail=100 | grep -q "Server startup complete" && echo true || echo false); \
		RABBIT2_STARTED=$$(kubectl logs -n $(RABBIT_NAMESPACE) -l app=rabbitmq-2 --tail=100 | grep -q "Server startup complete" && echo true || echo false); \
		if [ "$$RABBIT1_STARTED" = "true" ] && [ "$$RABBIT2_STARTED" = "true" ]; then \
			echo "$(GREEN)=== RabbitMQ servers started$(NC)"; \
			break; \
		fi; \
		echo "Waiting for rabbit server start... - $$((TIMEOUT - ELAPSED_TIME))s remaining"; \
		sleep 10; \
	done
	@echo "$(GREEN)--- Stage 5 END: RabbitMQ servers are ready$(NC)"
	@echo ""

# Install Kafka
.PHONY: install-kafka
install-kafka:
	@echo "$(BLUE)=== STAGE 6: Kafka Installation ===$(NC)"
	@echo "$(CYAN)--- Stage 6 BEGIN: Installing Kafka...$(NC)"
ifeq ($(CREATE_NAMESPACE),true)
	@kubectl get namespace $(KAFKA_NAMESPACE) || kubectl create namespace $(KAFKA_NAMESPACE)
endif
	@kubectl apply -f charts/kafka.yaml -n $(KAFKA_NAMESPACE) || \
		(echo "$(RED)Error: Kafka installation failed$(NC)" && echo "$(CYAN)Please check the helm output above for details$(NC)" && exit 1)
	@echo "$(GREEN)--- Stage 6 END: Kafka installation completed$(NC)"
	@echo ""

# Wait for Kafka server started
.PHONY: wait-for-kafka-server-started
wait-for-kafka-server-started:
	@echo "$(BLUE)=== STAGE 7: Wait for Kafka server started  ===$(NC)"
	@echo "$(CYAN)--- Stage 7 BEGIN: Waiting for kafka server started (timeout: 5 minutes)...$(NC)"
	@TIMEOUT=300; START_TIME=$$(date +%s); \
	while true; do \
		CURRENT_TIME=$$(date +%s); \
		ELAPSED_TIME=$$((CURRENT_TIME - START_TIME)); \
		if [ $$ELAPSED_TIME -ge $$TIMEOUT ]; then \
			echo "Timeout reached after 5 minutes. Started message not found in logs."; \
			exit 1; \
		fi; \
		KAFKA1_STARTED=$$(kubectl logs -n $(KAFKA_NAMESPACE) -l app=kafka-1 --tail=100 | grep -q "Kafka Server started" && echo true || echo false); \
		KAFKA2_STARTED=$$(kubectl logs -n $(KAFKA_NAMESPACE) -l app=kafka-2 --tail=100 | grep -q "Kafka Server started" && echo true || echo false); \
		if [ "$$KAFKA1_STARTED" = "true" ] && [ "$$KAFKA2_STARTED" = "true" ]; then \
			echo "$(GREEN)=== Kafka servers started$(NC)"; \
			break; \
		fi; \
		echo "Waiting for kafka server start... - $$((TIMEOUT - ELAPSED_TIME))s remaining"; \
		sleep 10; \
	done
	@echo "$(GREEN)--- Stage 7 END: Kafka servers are ready$(NC)"
	@echo ""

# Register RabbitMQ instances
.PHONY: register-rabbit-instances
register-rabbit-instances:
	@echo "$(BLUE)=== STAGE 8: Register RabbitMQ instances  ===$(NC)"
	@echo "$(CYAN)--- Stage 8 BEGIN: RabbitMQ instances registration...$(NC)"
	bash scripts/register-rabbit-instance-in-maas.sh rabbitmq-1
	bash scripts/register-rabbit-instance-in-maas.sh rabbitmq-2
	@echo "$(GREEN)--- Stage 8 END: RabbitMQ instances are registered$(NC)"
	@echo ""

# Register Kafka instances
register-kafka-instances:
	@echo "$(BLUE)=== STAGE 9: Register Kafka instances  ===$(NC)"
	@echo "$(CYAN)--- Stage 9 BEGIN: Kafka instances registration...$(NC)"
	bash scripts/register-kafka-instance-in-maas.sh kafka-1
	bash scripts/register-kafka-instance-in-maas.sh kafka-2
	@echo "$(GREEN)--- Stage 9 END: Kafka instances are registered$(NC)"
	@echo ""

# Main install target
.PHONY: install
install: validate show-config setup-repos \
	install-maas-service wait-for-maas-service-started \
	install-rabbitmq wait-for-rabbit-server-started \
	install-kafka wait-for-kafka-server-started \
	register-rabbit-instances register-kafka-instances
	@echo "$(GREEN)=== ALL STAGES COMPLETED SUCCESSFULLY ===$(NC)"


.PHONY: uninstall-maas-service
uninstall-maas-service:
	@echo "$(BLUE)=== STAGE 1: Uninstall MaaS service ===$(NC)"
	@echo "$(CYAN)--- Stage 1 BEGIN: Uninstalling MaaS service...$(NC)"
	@helm uninstall -n $(MAAS_NAMESPACE) maas-service 2>/dev/null || echo "MaaS service not found or already uninstalled"
	@echo "$(CYAN)--- Uninstalling MaaS service secrets created by scripts, not by helm...$(NC)"
	@kubectl delete secret maas-db-cipher-key-secret -n $(MAAS_NAMESPACE) 2>/dev/null || echo "Secret maas-db-cipher-key-secret not found or already deleted"
	@kubectl delete secret maas-db-postgresql-credentials-secret -n $(MAAS_NAMESPACE) 2>/dev/null || echo "Secret maas-db-postgresql-credentials-secret not found or already deleted"
	@echo "$(GREEN)--- Stage 1 END: MaaS service uninstalled$(NC)"
	@echo ""

.PHONY: delete-maas-database
delete-maas-database:
	@echo "$(BLUE)=== STAGE 2: Delete MaaS database ===$(NC)"
	@echo "$(CYAN)--- Stage 2 BEGIN: Deleting MaaS database...$(NC)"
	bash scripts/remove-maas-database-from-dbaas.sh
	@echo "$(GREEN)--- Stage 2 END: MaaS database deleted$(NC)"
	@echo ""

.PHONY: uninstall-rabbit
uninstall-rabbit:
	@echo "$(BLUE)=== STAGE 3: Uninstall RabbitMQ ===$(NC)"
	@echo "$(CYAN)--- Stage 3 BEGIN: Uninstalling RabbitMQ...$(NC)"
	@kubectl delete -f charts/rabbit.yaml -n $(RABBIT_NAMESPACE) 2>/dev/null || echo "RabbitMQ not found or already uninstalled"
	@echo "$(GREEN)--- Stage 3 END: RabbitMQ uninstalled$(NC)"
	@echo ""

.PHONY: uninstall-kafka
uninstall-kafka:
	@echo "$(BLUE)=== STAGE 4: Uninstall Kafka ===$(NC)"
	@echo "$(CYAN)--- Stage 4 BEGIN: Uninstalling Kafka...$(NC)"
	@kubectl delete -f charts/kafka.yaml -n $(KAFKA_NAMESPACE) 2>/dev/null || echo "Kafka not found or already uninstalled"
	@echo "$(GREEN)--- Stage 4 END: Kafka uninstalled$(NC)"
	@echo ""

# Main uninstall target
.PHONY: uninstall
uninstall: show-config uninstall-maas-service delete-maas-database uninstall-rabbit uninstall-kafka
	@echo "$(GREEN)=== ALL COMPONENTS UNINSTALLED SUCCESSFULLY ===$(NC)"

# Clean repositories
.PHONY: clean
clean:
	@echo "$(CYAN)--- Cleaning up repositories...$(NC)"
	@rm -rf repos/qubership-maas
	@echo "$(GREEN)--- Repositories cleaned$(NC)"