# Dependency versions for cloud-core-local-dev.
#
# Each variable pins the ref of a platform-level dependency (as opposed to the core services
# themselves, see local.mk) that this bootstrap line is designed to work with. An lts/* branch of
# core-bootstrap should pin these to whatever it shipped with, instead of tracking each
# dependency's main.
#
# A ref is resolved in this order, see "DEPENDENCY REFS" in the Makefile:
#   1. a value set from outside (make VAR=..., or the environment);
#   2. TEST_BRANCH, if it is set, differs from BASELINE_BRANCH and the dependency's repository has
#      a branch of that name;
#   3. the pin below.
# The pin, not BASELINE_BRANCH, is the fallback: these repositories have no lts/* branches.
#
# A pin and an explicit value are a branch or a tag, not a commit sha. TEST_BRANCH is a branch only.
#
# To take a dependency from main in a pipeline, pass it on the command line instead of editing this
# file, for example a nightly run: DBAAS_REPO_BRANCH=main ISTIO_REPO_BRANCH=main
# CORE_MESH_CONFIG_REPO_BRANCH=main MAAS_BRANCH=main.

# -----------------------------------------------------------------------------
# DBaaS
# -----------------------------------------------------------------------------

# qubership-dbaas: the bootstrap scripts and charts, and the image tag of dbaas-aggregator, which
# follows the ref. qubership-dbaas is released with semver tags, so the pin is a release tag.
DBAAS_REPO_BRANCH ?= v6.15.1

# -----------------------------------------------------------------------------
# MaaS
# -----------------------------------------------------------------------------

# qubership-maas: the charts, and the MaaS image tag, which the maas makefile derives from the ref:
# latest for main, the tag itself for a tag such as v5.6.2, <branch>-snapshot for another branch.
MAAS_BRANCH ?= v5.6.2

# -----------------------------------------------------------------------------
# Istio
# -----------------------------------------------------------------------------

# qubership-istio: the Istio distribution and configuration. Its release tags have no v prefix.
ISTIO_REPO_BRANCH ?= 1.2.1

# -----------------------------------------------------------------------------
# Core mesh config
# -----------------------------------------------------------------------------

# qubership-core-mesh-config: the mesh configuration.
CORE_MESH_CONFIG_REPO_BRANCH ?= v1.0.6
