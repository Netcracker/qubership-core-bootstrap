# Core Mesh Config profile: aws

ISTIO_NAMESPACE ?= istio-system
# Comma-separated list of namespaces to enable mesh on, e.g. "ns1,ns2"
MESH_NAMESPACES ?= core
RUN_SMOKE_TEST ?= true

#===============================================
# Core Mesh Config repository settings
CORE_MESH_CONFIG_REPO_URL ?= https://github.com/Netcracker/qubership-core-mesh-config.git
CORE_MESH_CONFIG_REPO_BRANCH ?= main
MESH_HELM_RELEASE_NAME = core-istio-mesh
# Extra helm args (optional), e.g.: MESH_HELM_EXTRA_ARGS = --set someKey=someValue
MESH_HELM_EXTRA_ARGS ?=
