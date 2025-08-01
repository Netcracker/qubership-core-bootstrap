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
MAAS_CONFIG_FILE ?= local.mk

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
CORE_BOOTSTRAP_IMAGE ?= ghcr.io/netcracker/core-bootstrap:latest 
CORE_CONFIG_CONSUL_ENABLED ?= false

# Components values
FACADE_OPERATOR_TAG ?= latest

INGRESS_GATEWAY_TAG ?= main-snapshot
INGRESS_GATEWAY_CLOUD_PUBLIC_HOST ?= svc.cluster.local
INGRESS_GATEWAY_CLOUD_PRIVATE_HOST ?= svc.cluster.local

CONTROL_PLANE_TAG ?= latest

PAAS_MEDIATION_TAG ?= latest

DBAAS_AGENT_TAG ?= latest

CORE_OPERATOR_TAG ?= feature-docker-image-build-snapshot
CORE_OPERATOR_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-core-operator

CONFIG_SERVER_TAG ?= feature-docker-image-build-snapshot
CONFIG_SERVER_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-config-server
CONFIG_SERVER_CONSUL_ENABLED ?= false

SITE_MANAGEMENT_TAG ?= feature-local-deployment-snapshot
