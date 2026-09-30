# Dependency versions for cloud-core-local-dev: which ref of each platform-level dependency (as
# opposed to the core services themselves, see TEST_BRANCH in local.mk) this bootstrap line is
# designed to work with. An lts/* branch of core-bootstrap should pin these to whatever it shipped
# with, instead of always tracking each dependency's main.
#
# ISTIO_REPO_BRANCH and CORE_MESH_CONFIG_REPO_BRANCH each accept a branch or a tag, not a commit sha:
# the istio and core-mesh-config makefiles clone them with git clone -b. TEST_BRANCH, unlike these
# two, is a branch only.
#
# Override any single value via: make VAR=value install - command-line assignments win over the
# ?= defaults below regardless of this file. A nightly pipeline that wants main/latest across the
# board passes ISTIO_REPO_BRANCH=main CORE_MESH_CONFIG_REPO_BRANCH=main
# MAAS_TAG=latest this way, instead of editing this file.

# Image/helm tag for MaaS. "latest" checks out main; otherwise checks out the matching GitHub tag.
MAAS_TAG ?= v5.5.10

# Branch or tag of the qubership-istio repository (Istio distribution/config). A value set from
# outside wins; otherwise TEST_BRANCH is used when the repository has a branch of that name and
# TEST_BRANCH differs from BASELINE_BRANCH; otherwise the value below. See "DEPENDENCY REFS" in
# the Makefile.
ISTIO_REPO_BRANCH ?= main

# Branch or tag of the qubership-core-mesh-config repository. Resolved like ISTIO_REPO_BRANCH.
CORE_MESH_CONFIG_REPO_BRANCH ?= main
