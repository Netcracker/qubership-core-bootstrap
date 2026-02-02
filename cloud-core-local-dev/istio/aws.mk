# Istio profile: aws

# installation parameters
CREATE_NAMESPACE ?= false
SKIP_CRDS ?= ftrue
ISTIO_NAMESPACE ?= istio-system
# Comma-separated list of namespaces to enable mesh on, e.g. "ns1,ns2"
MESH_NAMESPACES ?=

#===============================================
# Repo settings (where to clone Istio deployer helm package from)
ISTIO_REPO_URL ?= https://github.com/Netcracker/qubership-istio-distr.git
ISTIO_REPO_BRANCH ?= main

#===============================================
# Gateway API CRDs
GATEWAY_API_VERSION ?= v1.4.0
GATEWAY_API_CHANNEL ?= standard

#===============================================
# Istio deployer Helm release settings
ISTIO_RELEASE_NAME = istio-deployer
# Extra helm args (optional), e.g.: ISTIO_HELM_EXTRA_ARGS = --set someKey=someValue
ISTIO_HELM_EXTRA_ARGS ?=

#===============================================
# Core Istio Mesh Helm release settings
MESH_HELM_RELEASE_NAME = core-istio-mesh
# Example: MESH_HELM_CHART_PATH = ../some-chart
MESH_HELM_CHART_PATH ?= ../../core-istio-mesh/charts/istio-components
# Extra helm args (optional), e.g.: MESH_HELM_EXTRA_ARGS = --set someKey=someValue
MESH_HELM_EXTRA_ARGS ?=

