# Istio profile: local

#===============================================
# Namespace settings
CREATE_NAMESPACE ?= true
ISTIO_NAMESPACE ?= istio-system
SKIP_CRDS ?= false

#===============================================
# Gateway API CRDs
GATEWAY_API_VERSION ?= v1.4.0
GATEWAY_API_CHANNEL ?= standard

#===============================================
# Istio chart source
#
# image (default): take the qubership-istio chart, with the upstream Istio
#   sub-charts already vendored into charts/, from the qubership-istio-transfer
#   image that qubership-istio CI publishes. Nothing is fetched from
#   istio-release.storage.googleapis.com at install time.
# git: clone qubership-istio and run helm dependency update, which does fetch
#   the sub-charts from upstream. For working on the chart itself.
ISTIO_CHART_SOURCE ?= image
ISTIO_IMAGE ?= ghcr.io/netcracker/qubership-istio-transfer
# Minor Istio version the image tags are prefixed with. qubership-istio CI tags
# <minor>-latest on main, <minor>-<branch with / as _> on other branches and
# <x.y.z> on releases.
ISTIO_MINOR_VERSION ?= 1.30
# Image tag to use. Empty derives it from ISTIO_REPO_BRANCH, so the branch a
# caller already passes selects the image built from that branch.
ISTIO_IMAGE_TAG ?=
# Repository settings, used with ISTIO_CHART_SOURCE=git and to derive the tag
ISTIO_REPO_URL ?= https://github.com/Netcracker/qubership-istio.git
ISTIO_REPO_BRANCH ?= main
ISTIO_RELEASE_NAME = qubership-istio
# Extra helm args (optional), e.g.: ISTIO_HELM_EXTRA_ARGS = --set someKey=someValue
ISTIO_HELM_EXTRA_ARGS ?=
MONITORING_ENABLED ?= false