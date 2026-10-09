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
# The agents carry the legacy M2M calls of the core services to DBaaS and MaaS. In the k8s mode the
# core services call DBaaS and MaaS directly, so the agents are left out. The MaaS agent follows
# INSTALL_MAAS otherwise, and can be installed without it, for a MaaS that is not installed here.
INSTALL_DBAAS_AGENT ?= $(if $(filter k8s,$(M2M_AUTH_MODE)),false,true)
INSTALL_MAAS_AGENT ?= $(if $(filter k8s,$(M2M_AUTH_MODE)),false,$(INSTALL_MAAS))

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
# M2M authentication mode of the core services: legacy, hybrid (k8s with a fallback to legacy) or k8s.
# DBaaS and MaaS take the same mode unless it is set for them, which tests a core and its
# dependencies in different modes.
M2M_AUTH_MODE ?= legacy
DBAAS_M2M_AUTH_MODE ?= $(M2M_AUTH_MODE)
MAAS_M2M_AUTH_MODE ?= $(M2M_AUTH_MODE)
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
#   ingress-gateway (*) INGRESS_GATEWAY_BRANCH    INGRESS_GATEWAY_TAG
#   control-plane       CONTROL_PLANE_BRANCH      CONTROL_PLANE_TAG
#   paas-mediation      PAAS_MEDIATION_BRANCH     PAAS_MEDIATION_TAG
#   dbaas-agent         DBAAS_AGENT_BRANCH        DBAAS_AGENT_TAG
#   maas-agent          MAAS_AGENT_BRANCH         MAAS_AGENT_TAG
#   core-operator       CORE_OPERATOR_BRANCH      CORE_OPERATOR_TAG
#   config-server       CONFIG_SERVER_BRANCH      CONFIG_SERVER_TAG
#   site-management     SITE_MANAGEMENT_BRANCH    SITE_MANAGEMENT_TAG
#   core-bootstrap      CORE_BOOTSTRAP_BRANCH     CORE_BOOTSTRAP_TAG
#   cr-synchronizer     CR_SYNCHRONIZER_BRANCH    CR_SYNCHRONIZER_TAG
#
# The last two are images built from the core-bootstrap repository: nothing is cloned for them, only
# the image tag follows the branch.
#
# (*) ingress-gateway is the exception to BASELINE_BRANCH: its repository has no branch for each
# release line (no lts/*), so its baseline is always main, whatever BASELINE_BRANCH is. TEST_BRANCH
# and INGRESS_GATEWAY_BRANCH still apply to it.
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
#   maas                MAAS_BRANCH                     TAG, derived in the maas makefile
#
# These refs are a branch or a tag, not a commit sha.
#
# Where each value comes from (set explicitly, TEST_BRANCH, BASELINE_BRANCH, a pin or derived) is
# printed as a Markdown table by "make resolved-refs", which install runs first. RESOLVED_REFS_FILE
# names a file to append the table to, for example $GITHUB_STEP_SUMMARY.
#
# DBAAS_TAG is either set explicitly or follows DBAAS_REPO_BRANCH. The MaaS image tag follows
# MAAS_BRANCH the same way, and is derived by the maas makefile. In both cases the tag is latest for
# main, <branch>-snapshot for another branch, and the tag itself for a tag such as v6.15.1.

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
CONFIG_SERVER_CONSUL_ENABLED ?= false

# -----------------------------------------------------------------------------
# Images
# -----------------------------------------------------------------------------
#
# The image of a service is <SERVICE>_IMAGE_REPOSITORY:<SERVICE>_TAG, the tag being resolved as
# described in "Core service sources" above. The repository is passed to the service's chart as
# IMAGE_REPOSITORY. Before anything is installed, "make resolved-refs" prints the resolved
# branches and tags, then looks every image up in ghcr.io and marks the ones that do not exist; the
# install stops after that report if any is missing. CHECK_IMAGES=false turns the lookup off, for
# example for images that are not public.

CHECK_IMAGES ?= true

# Core services
FACADE_OPERATOR_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-facade-operator
INGRESS_GATEWAY_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-ingress-gateway
CONTROL_PLANE_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-control-plane
PAAS_MEDIATION_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-paas-mediation
DBAAS_AGENT_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-dbaas-agent
MAAS_AGENT_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-maas-agent
CORE_OPERATOR_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-core-operator
CONFIG_SERVER_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-config-server
SITE_MANAGEMENT_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-core-site-management

# Images built from the core-bootstrap repository. The full image name can be overridden as a
# whole with CORE_BOOTSTRAP_IMAGE and CR_SYNCHRONIZER_IMAGE.
CORE_BOOTSTRAP_IMAGE_REPOSITORY ?= ghcr.io/netcracker/core-bootstrap
CORE_BOOTSTRAP_IMAGE ?= $(CORE_BOOTSTRAP_IMAGE_REPOSITORY):$(CORE_BOOTSTRAP_TAG)
CR_SYNCHRONIZER_IMAGE_REPOSITORY ?= ghcr.io/netcracker/cr-synchronizer
CR_SYNCHRONIZER_IMAGE ?= $(CR_SYNCHRONIZER_IMAGE_REPOSITORY):$(CR_SYNCHRONIZER_TAG)

# DBaaS and MaaS install their own images, from the value files of their makefiles. These are the
# repositories those files use, listed here only for the image check.
DBAAS_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-dbaas
DBAAS_VALIDATION_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-dbaas-validation-image
MAAS_IMAGE_REPOSITORY ?= ghcr.io/netcracker/qubership-maas

# -----------------------------------------------------------------------------
# Core bootstrap
# -----------------------------------------------------------------------------

CORE_CONFIG_CONSUL_ENABLED ?= false
CORE_CONFIG_MAAS_ENABLED ?= false
# follows MAAS_NAMESPACE: MaaS is reached in the namespace it is installed in
CORE_CONFIG_MAAS_INTERNAL_ADDRESS ?= http://maas-service.$(MAAS_NAMESPACE):8080
