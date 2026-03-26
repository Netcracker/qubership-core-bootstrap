# Istio profile: aws

#===============================================
# Namespace settings
CREATE_NAMESPACE ?= false
ISTIO_NAMESPACE ?= istio-system

#===============================================
# Gateway API CRDs
SKIP_CRDS ?= true
GATEWAY_API_VERSION ?= v1.4.0
GATEWAY_API_CHANNEL ?= standard

#===============================================
# Istio chart repository settings
ISTIO_REPO_URL ?= https://github.com/Netcracker/qubership-istio-distr.git
ISTIO_REPO_BRANCH ?= main
ISTIO_RELEASE_NAME = qubership-istio
# Extra helm args (optional), e.g.: ISTIO_HELM_EXTRA_ARGS = --set someKey=someValue
ISTIO_HELM_EXTRA_ARGS ?=
