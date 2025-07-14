# values files for helm packages
MAAS_VALUES_FILE ?= ./maas-values.yaml
MAAS_PROFILE_FILE ?= ./repos/qubership-maas/helm-templates/maas-service/resource-profiles/dev.yaml

# namespace parameters
MAAS_NAMESPACE ?= maas
RABBIT_NAMESPACE ?= rabbit
KAFKA_NAMESPACE ?= kafka

# maas parameters
SERVICE_NAME ?= maas-service
TAG ?= latest

export MAAS_VALUES_FILE
export MAAS_PROFILE_FILE
export MAAS_NAMESPACE
export RABBIT_NAMESPACE
export KAFKA_NAMESPACE
export SERVICE_NAME
export TAG

# installation parameters - not propagated to helm values
CREATE_NAMESPACE ?= true