# Configuration file for cloud-core-local-dev
# This file contains configurable values that can be overridden

# -----------------------------------------------------------------------------
# Installation
# -----------------------------------------------------------------------------

# Parameters used by the script, but not propagated to helm values
CREATE_NAMESPACE ?= true
INSTALL_CRDS ?= true
INSTALL_METRICS_SERVER ?= true
INSTALL_MONITORING ?= false
INSTALL_CONSUL ?= true
CONSUL_ACLS ?= false
INSTALL_DBAAS ?= true
INSTALL_MAAS ?= false
INSTALL_ISTIO ?= false

# Config files of the components installed by their own makefiles. A relative path is resolved
# against the component's folder.
DBAAS_CONFIG_FILE ?= local.mk
MAAS_CONFIG_FILE ?= local.mk
ISTIO_CONFIG_FILE ?= local.mk
MESH_CONFIG_FILE ?= local.mk

# -----------------------------------------------------------------------------
# Namespaces
# -----------------------------------------------------------------------------

CORE_NAMESPACE ?= core
ORIGIN_NAMESPACE ?= ${CORE_NAMESPACE}
CONSUL_NAMESPACE ?= consul
MONITORING_NAMESPACE ?= monitoring
# Propagated to dbaas-install
PG_NAMESPACE ?= postgres
DBAAS_NAMESPACE ?= dbaas
# Propagated to maas-install
MAAS_NAMESPACE ?= maas
RABBIT_NAMESPACE ?= rabbit
KAFKA_NAMESPACE ?= kafka
# Propagated to istio-install
ISTIO_NAMESPACE ?= istio-system

# -----------------------------------------------------------------------------
# General values
# -----------------------------------------------------------------------------

DEPLOYMENT_SESSION_ID ?= cloud-core-local-dev
SERVICE_MESH_TYPE ?= Core
KUBERNETES_M2M_ENABLED ?= false
MONITORING_ENABLED ?= false
CONSUL_ENABLED ?= true
CONSUL_SERVICE_NAME ?= consul-consul-server

# -----------------------------------------------------------------------------
# DBaaS
# -----------------------------------------------------------------------------

DBAAS_SERVICE_NAME ?= dbaas-aggregator

# -----------------------------------------------------------------------------
# MaaS
# -----------------------------------------------------------------------------

KAFKA_INSTANCES ?= kafka-1
# An empty value skips the rabbit installation
RABBIT_INSTANCES ?= 

# -----------------------------------------------------------------------------
# Core service sources
# -----------------------------------------------------------------------------
#
# Two more sets of values are resolved from the branches below, and have no defaults in this file:
#   - the branch and the image tag of each service, <SERVICE>_BRANCH and <SERVICE>_TAG (for example
#     CONFIG_SERVER_BRANCH): see "CORE SERVICE BRANCHES AND IMAGE TAGS" in the main Makefile;
#   - the refs of the dependencies (ISTIO_REPO_BRANCH, DBAAS_REPO_BRANCH and
#     CORE_MESH_CONFIG_REPO_BRANCH): see dependencies.mk.

# TEST_BRANCH: the branch of the core service repositories to test. It has no default: when it is
# not set, only BASELINE_BRANCH is used. A tag or a commit sha is not accepted.
#
# Fallback when a repository has no TEST_BRANCH branch: main, or a release line such as lts/26.3
# when TEST_BRANCH itself is being tested against that line.
BASELINE_BRANCH ?= main

# -----------------------------------------------------------------------------
# Core services
# -----------------------------------------------------------------------------

INGRESS_GATEWAY_CLOUD_PUBLIC_HOST ?= svc.cluster.local
INGRESS_GATEWAY_CLOUD_PRIVATE_HOST ?= svc.cluster.local

MAAS_AGENT_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-maas-agent

CORE_OPERATOR_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-core-operator

CONFIG_SERVER_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-config-server
CONFIG_SERVER_CONSUL_ENABLED ?= false

# -----------------------------------------------------------------------------
# Core bootstrap
# -----------------------------------------------------------------------------

CORE_BOOTSTRAP_IMAGE ?= ghcr.io/netcracker/core-bootstrap:latest 
CORE_CONFIG_CONSUL_ENABLED ?= false
CORE_CONFIG_MAAS_ENABLED ?= false
CORE_CONFIG_MAAS_INTERNAL_ADDRESS ?= http://maas-service.maas:8080
