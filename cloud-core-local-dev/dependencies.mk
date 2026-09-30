# Dependency versions for cloud-core-local-dev: which ref of each platform-level dependency (as
# opposed to the core services themselves, see TEST_BRANCH in local.mk) this bootstrap line is
# designed to work with. An lts/* branch of core-bootstrap should pin these to whatever it shipped
# with, instead of always tracking each dependency's main.
#
# ISTIO_REPO_BRANCH, CORE_MESH_CONFIG_REPO_BRANCH, DBAAS_REPO_BRANCH and MAAS_BRANCH each accept a
# branch or a tag, not a commit sha. TEST_BRANCH, unlike these, is a branch only.
#
# Override any single value via: make VAR=value install - command-line assignments win over the
# ?= defaults below regardless of this file. A nightly pipeline that wants main/latest across the
# board passes ISTIO_REPO_BRANCH=main DBAAS_REPO_BRANCH=main CORE_MESH_CONFIG_REPO_BRANCH=main
# MAAS_BRANCH=main this way, instead of editing this file.

# Branch or tag of the qubership-maas repository, for its charts and for the MaaS image tag, which
# the maas makefile derives from it: latest for main, the tag itself for a tag such as v5.5.10,
# <branch>-snapshot for another branch. Resolved like DBAAS_REPO_BRANCH below.
MAAS_BRANCH ?= v5.5.10

# Branch or tag of the qubership-istio repository (Istio distribution/config). A value set from
# outside wins; otherwise TEST_BRANCH is used when the repository has a branch of that name and
# TEST_BRANCH differs from BASELINE_BRANCH; otherwise the value below. See "DEPENDENCY REFS" in
# the Makefile.
ISTIO_REPO_BRANCH ?= main

# Branch or tag of the qubership-core-mesh-config repository. Resolved like ISTIO_REPO_BRANCH.
CORE_MESH_CONFIG_REPO_BRANCH ?= main

# Branch or tag of the qubership-dbaas repository, for its bootstrap scripts and charts and for the
# image tag of dbaas-aggregator. Resolved like ISTIO_REPO_BRANCH, so this pin replaces
# BASELINE_BRANCH as the fallback: qubership-dbaas is released with semver tags and has no lts/*
# branches. Pin it to a release tag, e.g. v6.15.1, on a release line of this repository.
DBAAS_REPO_BRANCH ?= main
