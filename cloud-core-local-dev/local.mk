# Configuration file for cloud-core-local-dev
# This file contains configurable values that can be overridden

# Installation configuration
# parameters used by script, but not propagated to helm values
CREATE_NAMESPACE ?= true
INSTALL_CRDS ?= true
INSTALL_METRICS_SERVER ?= true
INSTALL_MONITORING ?= false
INSTALL_CONSUL ?= true
CONSUL_ACLS ?= false
INSTALL_DBAAS ?= true
# config file for dbaas installation - relative path will be resolved upon ./dbaas folder
DBAAS_CONFIG_FILE ?= local.mk
INSTALL_MAAS ?= false
# config file for maas installation - relative path will be resolved upon ./maas folder
MAAS_CONFIG_FILE ?= local.mk
INSTALL_ISTIO ?= false
# config file for istio installation - relative path will be resolved upon ./istio folder
ISTIO_CONFIG_FILE ?= local.mk
# config file for core-mesh-config - relative path will be resolved upon ./core-mesh-config folder
MESH_CONFIG_FILE ?= local.mk
# ISTIO_REPO_BRANCH and CORE_MESH_CONFIG_REPO_BRANCH: see dependencies.mk

# Namespace configuration
CORE_NAMESPACE ?= core
CONSUL_NAMESPACE ?= consul
MONITORING_NAMESPACE ?= monitoring
# pg & dbaas namespaces are propagated to dbaas-install
PG_NAMESPACE ?= postgres
DBAAS_NAMESPACE ?= dbaas
# below namespaces are propagated to maas-install
MAAS_NAMESPACE ?= maas
RABBIT_NAMESPACE ?= rabbit
KAFKA_NAMESPACE ?= kafka
# istio namespace is propagated to istio-install
ISTIO_NAMESPACE ?= istio-system

# General values
DEPLOYMENT_SESSION_ID ?= cloud-core-local-dev
MONITORING_ENABLED ?= false
CONSUL_SERVICE_NAME ?= consul-consul-server
CONSUL_ENABLED ?= true

# DBaaS configuration
DBAAS_SERVICE_NAME ?= dbaas-aggregator

# MaaS configuration
KAFKA_INSTANCES ?= kafka-1
# empty value - skip rabbit installation
RABBIT_INSTANCES ?= 

# Core bootstrap configuration
CORE_BOOTSTRAP_IMAGE ?= ghcr.io/netcracker/core-bootstrap:latest 
CORE_CONFIG_CONSUL_ENABLED ?= false
CORE_CONFIG_MAAS_ENABLED ?= false
CORE_CONFIG_MAAS_INTERNAL_ADDRESS ?= http://maas-service.maas:8080

# Components values
# *_TAG (incl. DBAAS_TAG, dbaas-aggregator's) defaults: see "CORE SERVICE BRANCHES AND IMAGE TAGS" in
# the main Makefile, resolved from TEST_BRANCH/BASELINE_BRANCH (below in this file) once this
# whole file has been included, rather than a flat default here.
INGRESS_GATEWAY_CLOUD_PUBLIC_HOST ?= svc.cluster.local
INGRESS_GATEWAY_CLOUD_PRIVATE_HOST ?= svc.cluster.local

MAAS_AGENT_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-maas-agent

CORE_OPERATOR_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-core-operator

CONFIG_SERVER_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-config-server
CONFIG_SERVER_CONSUL_ENABLED ?= false

SERVICE_MESH_TYPE ?= Core

# branch of the core service repositories to test; a tag or a commit sha is not accepted
# A single service can be pointed at another branch with <SERVICE>_BRANCH, e.g. CONFIG_SERVER_BRANCH:
# see "CORE SERVICE BRANCHES AND IMAGE TAGS" in the main Makefile.
TEST_BRANCH ?= main
# fallback when a repository has no TEST_BRANCH branch: main, or a release line such as lts/26.3
# when TEST_BRANCH itself is being tested against that line
BASELINE_BRANCH ?= main

ORIGIN_NAMESPACE ?= ${CORE_NAMESPACE}

KUBERNETES_M2M_ENABLED ?= false