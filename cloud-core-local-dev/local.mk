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
# The branch and the image tag of a single service can be set from outside (make VAR=..., or the
# environment) with the variables below. They have no defaults in this file: an unset branch is
# resolved from TEST_BRANCH and BASELINE_BRANCH, and an unset tag follows the branch the service is
# cloned at (latest for main, <branch>-snapshot for any other branch).
#
#   Service             Branch variable           Image tag variable
#   facade-operator     FACADE_OPERATOR_BRANCH    FACADE_OPERATOR_TAG
#   ingress-gateway     INGRESS_GATEWAY_BRANCH    INGRESS_GATEWAY_TAG
#   control-plane       CONTROL_PLANE_BRANCH      CONTROL_PLANE_TAG
#   paas-mediation      PAAS_MEDIATION_BRANCH     PAAS_MEDIATION_TAG
#   dbaas-agent         DBAAS_AGENT_BRANCH        DBAAS_AGENT_TAG
#   maas-agent          MAAS_AGENT_BRANCH         MAAS_AGENT_TAG
#   core-operator       CORE_OPERATOR_BRANCH      CORE_OPERATOR_TAG
#   config-server       CONFIG_SERVER_BRANCH      CONFIG_SERVER_TAG
#   site-management     SITE_MANAGEMENT_BRANCH    SITE_MANAGEMENT_TAG
#
# An explicit branch must exist in the service's repository, or make stops with an error.
# See "CORE SERVICE BRANCHES AND IMAGE TAGS" in the main Makefile.
#
# The components below fall back to the version pinned in dependencies.mk, not to BASELINE_BRANCH.
# Their ref is taken from TEST_BRANCH when it is set, differs from BASELINE_BRANCH and exists in the
# component's repository (not for maas), and an explicit value wins over both. See "DEPENDENCY REFS"
# in the main Makefile.
#
#   Component           Ref variable                    Image tag variable
#   dbaas-aggregator    DBAAS_REPO_BRANCH               DBAAS_TAG
#   istio               ISTIO_REPO_BRANCH               -
#   core-mesh-config    CORE_MESH_CONFIG_REPO_BRANCH    -
#   maas                -                               MAAS_TAG
#
# These refs are a branch or a tag, not a commit sha.
#
# DBAAS_TAG is either set explicitly or follows DBAAS_REPO_BRANCH: latest for main,
# <branch>-snapshot for another branch, and the tag itself for a tag such as v6.15.1.

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
