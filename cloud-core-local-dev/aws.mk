# Configuration file for cloud-core-local-dev
# This file contains configurable values that can be overridden

# Installation configuration
# parameters used by script, but not propagated to helm values
CREATE_NAMESPACE ?= false
INSTALL_CRDS ?= false
INSTALL_METRICS_SERVER ?= false
INSTALL_MONITORING ?= false
INSTALL_CONSUL ?= false
INSTALL_DBAAS ?= true
# config file for dbaas installation - relative path will be resolved upon ./dbaas folder
DBAAS_CONFIG_FILE ?= aws.mk
INSTALL_MAAS ?= true
# config file for maas installation - relative path will be resolved upon ./maas folder
MAAS_CONFIG_FILE ?= aws.mk
INSTALL_ISTIO ?= false
# config file for istio installation - relative path will be resolved upon ./istio folder
ISTIO_CONFIG_FILE ?= aws.mk
# config file for core-mesh-config - relative path will be resolved upon ./core-mesh-config folder
MESH_CONFIG_FILE ?= aws.mk
# ISTIO_REPO_BRANCH and CORE_MESH_CONFIG_REPO_BRANCH: see dependencies.mk

# Namespace configuration
CORE_NAMESPACE ?= core-1-core
CONSUL_NAMESPACE ?= consul
MONITORING_NAMESPACE ?= monitoring
# pg & dbaas namespaces are propagated to dbaas-install
PG_NAMESPACE ?= core-1-postgres
DBAAS_NAMESPACE ?= core-1-dbaas
# below namespaces are propagated to maas-install
MAAS_NAMESPACE ?= core-1-maas
RABBIT_NAMESPACE ?= core-1-maas
KAFKA_NAMESPACE ?= core-1-maas
# istio namespace is propagated to istio-install
ISTIO_NAMESPACE ?= istio-system

# General values
DEPLOYMENT_SESSION_ID ?= cloud-core-aws-dev
MONITORING_ENABLED ?= false
CONSUL_SERVICE_NAME ?= consul-server
CONSUL_ENABLED ?= true

# DBaaS configuration
DBAAS_SERVICE_NAME ?= dbaas-aggregator

# MaaS configuration
KAFKA_INSTANCES ?= kafka-1 kafka-2
# empty value - skip rabbit installation
RABBIT_INSTANCES ?= rabbit-1

# Core bootstrap configuration
CORE_BOOTSTRAP_IMAGE_REPOSITORY ?= ghcr.io/netcracker/core-bootstrap
CORE_BOOTSTRAP_IMAGE ?= $(CORE_BOOTSTRAP_IMAGE_REPOSITORY):$(CORE_BOOTSTRAP_TAG)
CR_SYNCHRONIZER_IMAGE_REPOSITORY ?= ghcr.io/netcracker/cr-synchronizer
CR_SYNCHRONIZER_IMAGE ?= $(CR_SYNCHRONIZER_IMAGE_REPOSITORY):$(CR_SYNCHRONIZER_TAG)
CORE_CONFIG_CONSUL_ENABLED ?= false
CORE_CONFIG_MAAS_ENABLED ?= true
CORE_CONFIG_MAAS_INTERNAL_ADDRESS ?= http://maas-service.maas:8080

# Components values
FACADE_OPERATOR_TAG ?= latest

INGRESS_GATEWAY_TAG ?= latest
INGRESS_GATEWAY_CLOUD_PUBLIC_HOST ?= svc.cluster.local
INGRESS_GATEWAY_CLOUD_PRIVATE_HOST ?= svc.cluster.local

CONTROL_PLANE_TAG ?= latest

PAAS_MEDIATION_TAG ?= latest

DBAAS_AGENT_TAG ?= latest

MAAS_AGENT_TAG ?= latest

CORE_OPERATOR_TAG ?= latest

CONFIG_SERVER_TAG ?= latest
CONFIG_SERVER_CONSUL_ENABLED ?= false

SITE_MANAGEMENT_TAG ?= latest

# Images: see local.mk
CHECK_IMAGES ?= true
FACADE_OPERATOR_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-facade-operator
INGRESS_GATEWAY_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-ingress-gateway
CONTROL_PLANE_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-control-plane
PAAS_MEDIATION_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-paas-mediation
DBAAS_AGENT_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-dbaas-agent
MAAS_AGENT_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-maas-agent
CORE_OPERATOR_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-core-operator
CONFIG_SERVER_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-config-server
SITE_MANAGEMENT_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-site-management
DBAAS_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-dbaas
DBAAS_VALIDATION_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-dbaas-validation-image
MAAS_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-maas

SERVICE_MESH_TYPE ?= Core

ORIGIN_NAMESPACE ?= ${CORE_NAMESPACE}

KUBERNETES_M2M_ENABLED ?= false