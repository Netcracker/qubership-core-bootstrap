# Dependency versions for cloud-core-local-dev: which ref of each platform-level dependency (as
# opposed to the core services themselves, see TEST_BRANCH in local.mk) this bootstrap line is
# designed to work with. An lts/* branch of core-bootstrap should pin these to whatever it shipped
# with, instead of always tracking each dependency's main.
#
# The *_REPO_BRANCH variables (despite the name, kept for consistency with TEST_BRANCH) each
# accept a branch, tag or commit sha; see clone_or_update_repo. A ref that does not resolve on the
# remote falls back to BASELINE_BRANCH (local.mk), then to the remote's default branch.
#
# Override any single value via: make VAR=value install - command-line assignments win over the
# ?= defaults below regardless of this file. A nightly pipeline that wants main/latest across the
# board passes ISTIO_REPO_BRANCH=main DBAAS_REPO_BRANCH=main CORE_MESH_CONFIG_REPO_BRANCH=main
# MAAS_TAG=latest this way, instead of editing this file.

# Image/helm tag for MaaS. "latest" checks out main; otherwise checks out the matching GitHub tag.
MAAS_TAG ?= v5.5.10

# Branch, tag or commit of the qubership-istio-distr repository (Istio distribution/config).
ISTIO_REPO_BRANCH ?= main

# Branch, tag or commit of the qubership-core-mesh-config repository.
CORE_MESH_CONFIG_REPO_BRANCH ?= main

# Branch, tag or commit of the qubership-dbaas repository's bootstrap scripts and charts. Defaults
# to tracking TEST_BRANCH, since a caller testing a coordinated qubership-dbaas branch alongside
# the core services under TEST_BRANCH expects it to be picked up the same way; set explicitly here
# or on the command line to pin it independently of TEST_BRANCH.
DBAAS_REPO_BRANCH ?= $(TEST_BRANCH)
