# values files for helm packages
MAAS_VALUES_FILE ?= ./maas-values.yaml
MAAS_PROFILE_FILE ?= ./repos/qubership-maas/helm-templates/maas-service/resource-profiles/dev.yaml

# namespace parameters
MAAS_NAMESPACE ?= maas
RABBIT_NAMESPACE ?= rabbit
KAFKA_NAMESPACE ?= kafka
DBAAS_NAMESPACE ?= dbaas

# maas parameters
TAG ?= latest

#dbaas parameters
DBAAS_AGGREGATOR_ADDRESS ?= http://dbaas-aggregator.${DBAAS_NAMESPACE}.svc.cluster.local:8080

# maas credentials
MAAS_ACCOUNT_MANAGER_USERNAME ?= manager
MAAS_ACCOUNT_MANAGER_PASSWORD ?= manager

MAAS_DEPLOYER_CLIENT_USERNAME ?= client
MAAS_DEPLOYER_CLIENT_PASSWORD ?= client

# dbaas credentials
DBAAS_CLUSTER_DBA_CREDENTIALS_USERNAME ?= cluster-dba
DBAAS_CLUSTER_DBA_CREDENTIALS_PASSWORD ?= password

export MAAS_VALUES_FILE
export MAAS_PROFILE_FILE
export MAAS_NAMESPACE
export RABBIT_NAMESPACE
export KAFKA_NAMESPACE
export DBAAS_NAMESPACE
export DBAAS_AGGREGATOR_ADDRESS
export TAG
export MAAS_ACCOUNT_MANAGER_USERNAME
export MAAS_ACCOUNT_MANAGER_PASSWORD
export MAAS_DEPLOYER_CLIENT_USERNAME
export MAAS_DEPLOYER_CLIENT_PASSWORD
export DBAAS_CLUSTER_DBA_CREDENTIALS_USERNAME
export DBAAS_CLUSTER_DBA_CREDENTIALS_PASSWORD

# installation parameters - not propagated to helm values
CREATE_NAMESPACE ?= true