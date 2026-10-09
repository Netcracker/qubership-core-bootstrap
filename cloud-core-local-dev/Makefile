# =============================================================================
# PHONY TARGETS
# =============================================================================

# Main targets
.PHONY: all install help

# Installation targets
.PHONY: deploy deploy-dependencies deploy-core \
	clone-repos template-values-files \
	deploy-consul deploy-consul-old deploy-consul-print-start install-consul wait-for-consul check-consul-connectivity \
	deploy-monitoring deploy-metrics-server \
	deploy-istio \
	deploy-core-mesh-config \
	deploy-dbaas \
	deploy-maas \
	prepare-core-bootstrap prepare-core-bootstrap-print-start deploy-cloud-core-configuration \
	deploy-cloud-core-components deploy-cloud-core-print-start \
	apply-facade-operator-crds deploy-facade-operator \
	deploy-ingress-gateway \
	deploy-site-management \
	deploy-paas-mediation \
	deploy-control-plane \
	deploy-dbaas-agent \
	deploy-maas-agent \
	apply-core-operator-crds deploy-core-operator \
	deploy-config-server

# Uninstallation targets
.PHONY: uninstall uninstall-core uninstall-dependencies \
	uninstall-cloud-core-components uninstall-cloud-core-configuration \
	uninstall-consul uninstall-monitoring uninstall-metrics-server \
	uninstall-istio \
	uninstall-core-mesh-config \
	uninstall-dbaas \
	uninstall-maas \
	uninstall-control-plane-db-credentials \
	uninstall-facade-operator uninstall-ingress-gateway uninstall-site-management \
	uninstall-paas-mediation uninstall-control-plane uninstall-dbaas-agent uninstall-maas-agent \
	uninstall-core-operator uninstall-config-server \
	cleanup-crds cleanup-namespaces

# Validation and utility targets
.PHONY: validate validate-cli validate-crd \
	ensure-core-namespace validate-consul-namespace \
	change-consul-values clone-consul-repo \
	mesh-smoke-test cleanup-mesh-test

# =============================================================================
# CONFIGURATION
# =============================================================================

# Note which dependency refs the caller sets from outside (command line or environment): the files
# included below only ever give them defaults. See "DEPENDENCY REFS" further down.
ifeq ($(origin ISTIO_REPO_BRANCH),undefined)
ISTIO_REPO_BRANCH_UNSET := true
endif
ifeq ($(origin CORE_MESH_CONFIG_REPO_BRANCH),undefined)
CORE_MESH_CONFIG_REPO_BRANCH_UNSET := true
endif
ifeq ($(origin DBAAS_REPO_BRANCH),undefined)
DBAAS_REPO_BRANCH_UNSET := true
endif
ifeq ($(origin MAAS_BRANCH),undefined)
MAAS_BRANCH_UNSET := true
endif

# Load configuration from .mk file
CONFIG_FILE ?= local.mk
include $(CONFIG_FILE)

# Load dependency versions (tags/refs). Profiles may override DEPENDENCIES_FILE.
DEPENDENCIES_FILE ?= dependencies.mk
include $(DEPENDENCIES_FILE)

# Ensure recipes (incl. envsubst) see Make variables in the environment.
# This includes config-loaded vars and `make VAR=...` overrides.
.EXPORT_ALL_VARIABLES:

# Repository directory
REPOS_DIR ?= ./.tmp

# Color definitions
YELLOW := \033[1;33m
GREEN := \033[1;32m
RED := \033[0;31m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m

CREATE_NAMESPACE_OPTION ?= $(if $(filter true,$(CREATE_NAMESPACE)), --create-namespace, )
CONSUL_ACLS_OPTION ?= $(if $(filter true,$(CONSUL_ACLS)), --set global.acls.manageSystemACLs=true, )

ifeq ($(DEBUG),true)
  HELM_DEBUG_FLAGS := --debug --dry-run
else
  HELM_DEBUG_FLAGS :=
endif

# =============================================================================
# BRANCH AND IMAGE TAG RESOLUTION
# =============================================================================
#
# Each core service repository is cloned at, and its image tag follows, the first of:
#   1. <SERVICE>_BRANCH, e.g. CONFIG_SERVER_BRANCH, when set from outside (make VAR=... or the
#      environment). The branch must exist in that repository, or make stops with an error;
#   2. TEST_BRANCH, if it is set and that repository has a branch of that name;
#   3. BASELINE_BRANCH; for ingress-gateway always main, whatever BASELINE_BRANCH is, as its
#      repository has no branch for each release line.
# The image tag is latest for main, and <branch, / replaced by ->-snapshot for any other branch:
# the rolling tag qubership-core-infra's shared docker-build.yaml publishes. An explicit
# <SERVICE>_TAG (command line or environment) always wins over the tag derived here.
# <SERVICE> is FACADE_OPERATOR, INGRESS_GATEWAY, CONTROL_PLANE, PAAS_MEDIATION, DBAAS_AGENT,
# MAAS_AGENT, CORE_OPERATOR, CONFIG_SERVER or SITE_MANAGEMENT. CORE_BOOTSTRAP and CR_SYNCHRONIZER
# follow the same rule, for the images built from the repository of this bootstrap; nothing is
# cloned for them. As a branch of that repository may have no image, the tag is also looked up in
# ghcr.io, and a TEST_BRANCH without an image falls back to BASELINE_BRANCH, per image.
#
# ISTIO_REPO_BRANCH, CORE_MESH_CONFIG_REPO_BRANCH, DBAAS_REPO_BRANCH and MAAS_BRANCH are resolved in
# this order:
#   1. a value set from outside (make VAR=... or the environment) is used as is;
#   2. otherwise TEST_BRANCH, if it is set and that branch exists in the dependency's repository;
#   3. otherwise the value pinned in dependencies.mk.
# Step 2 only applies when TEST_BRANCH differs from BASELINE_BRANCH. Equal values mean nothing
# is being tested on a separate branch, and the pin in dependencies.mk should win over
# TEST_BRANCH. Each pin and each explicit value is a branch or a tag, not a commit sha;
# TEST_BRANCH is a branch. These are the versions of qubership-dbaas, which is released with semver
# tags rather than by release line, and of qubership-maas, istio and core-mesh-config.
#
# DBAAS_TAG follows DBAAS_REPO_BRANCH: latest for main, <branch>-snapshot for another branch, and
# the tag itself (a release such as v6.15.1) for a tag. An explicit DBAAS_TAG always wins.
# MAAS_BRANCH is passed to the maas makefile, which clones qubership-maas at it and derives the
# MaaS image tag from it in the same way; see maas/Makefile.
#
# The resolved values and where each came from are reported by the resolved-refs target, which
# install runs first; see below.

# non-empty when repository $(1) has a branch named $(2); a tag or commit of that name does not count
define branch_exists
$(if $(2),$(strip $(shell git ls-remote --heads $(1) refs/heads/$(2) 2>/dev/null)))
endef

# image tag published for branch $(1)
define tag_for_branch
$(if $(filter main,$(1)),latest,$(subst /,-,$(1))-snapshot)
endef

# image tag for ref $(2) of repository $(1), a branch or a tag; anything else stops make
define ref_image_tag
$(if $(2),$(if $(call branch_exists,$(1),$(2)),$(call tag_for_branch,$(2)),$(if $(strip $(shell git ls-remote --tags $(1) refs/tags/$(2) 2>/dev/null)),$(2),$(error '$(2)' is not a branch or a tag of $(1)))))
endef

# Resolve the branch and the image tag of a core service; $(1) is the variable prefix, e.g.
# CONFIG_SERVER, and $(2) the URL of its repository. Sets <SERVICE>_CLONE_BRANCH, the branch the
# repository is cloned at, and <SERVICE>_TAG, with <SERVICE>_BRANCH_SOURCE and <SERVICE>_TAG_SOURCE
# telling where each came from. The optional $(3) is a fixed baseline branch of this service, used
# instead of BASELINE_BRANCH, for a repository that has no branch for each release line.
define core_service
$(1)_BASELINE := $$(if $(3),$(3),$$(BASELINE_BRANCH))
$(1)_BRANCH_SOURCE := $$(if $$($(1)_BRANCH),explicit,$$(if $$(call branch_exists,$(2),$$(TEST_BRANCH)),TEST_BRANCH,$$(if $(3),fixed baseline $(3),BASELINE_BRANCH)))
$(1)_CLONE_BRANCH := $$(if $$(filter explicit,$$($(1)_BRANCH_SOURCE)),$$(if $$(call branch_exists,$(2),$$($(1)_BRANCH)),$$($(1)_BRANCH),$$(error Branch '$$($(1)_BRANCH)' not found in $(2))),$$(if $$(filter TEST_BRANCH,$$($(1)_BRANCH_SOURCE)),$$(TEST_BRANCH),$$($(1)_BASELINE)))
ifeq ($$(origin $(1)_TAG),undefined)
$(1)_TAG_SOURCE := derived from the branch
$(1)_TAG := $$(call tag_for_branch,$$($(1)_CLONE_BRANCH))
else
$(1)_TAG_SOURCE := explicit
endif
endef

# Resolve a dependency ref; $(1) is the variable, e.g. ISTIO_REPO_BRANCH, and $(2) the URL of its
# repository. $(1)_SOURCE tells where the value came from. $(1)_UNSET was set before local.mk and
# dependencies.mk were included, when the caller had not set the variable.
define dependency_ref
$(1)_SOURCE := $$(if $$($(1)_UNSET),pin (dependencies.mk),explicit)
ifneq ($$(TEST_BRANCH),$$(BASELINE_BRANCH))
ifdef $(1)_UNSET
ifneq ($$(call branch_exists,$(2),$$(TEST_BRANCH)),)
$(1) := $$(TEST_BRANCH)
$(1)_SOURCE := TEST_BRANCH
endif
endif
endif
endef

# HTTP status of the manifest of tag $(2) of the image $(1) in ghcr.io; 404 means the tag does not
# exist. Empty for an image of another registry, or when the registry cannot be reached.
define ghcr_tag_status
$(if $(filter ghcr.io/%,$(1)),$(strip $(shell \
	path="$(patsubst ghcr.io/%,%,$(1))"; \
	token="$$(curl -fsS "https://ghcr.io/token?scope=repository:$$path:pull" 2>/dev/null | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"; \
	curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $$token" \
		-H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json' \
		"https://ghcr.io/v2/$$path/manifests/$(2)" 2>/dev/null)))
endef

# A branch can exist in the repository without an image: the images are built for main, lts
# branches, tags and pull requests that change them. For the image of $(1), e.g. CORE_BOOTSTRAP,
# resolved from a branch and not from an explicit tag or image name, check that the tag exists in
# ghcr.io. If it does not, a TEST_BRANCH falls back to BASELINE_BRANCH, and an explicit
# <SERVICE>_BRANCH stops make with an error.
define image_exists_or_baseline
ifneq ($$(filter derived%,$$($(1)_TAG_SOURCE)),)
ifneq ($$(filter-out BASELINE_BRANCH,$$($(1)_BRANCH_SOURCE)),)
ifeq ($$(origin $(1)_IMAGE),file)
ifeq ($$(call ghcr_tag_status,$$($(1)_IMAGE_REPOSITORY),$$($(1)_TAG)),404)
ifeq ($$($(1)_BRANCH_SOURCE),explicit)
$$(error $(1)_BRANCH '$$($(1)_BRANCH)': there is no image $$($(1)_IMAGE_REPOSITORY):$$($(1)_TAG))
endif
$(1)_CLONE_BRANCH := $$(BASELINE_BRANCH)
$(1)_BRANCH_SOURCE := BASELINE_BRANCH (no image for TEST_BRANCH)
$(1)_TAG := $$(call tag_for_branch,$$($(1)_CLONE_BRANCH))
endif
endif
endif
endif
endef

CORE_SERVICES := FACADE_OPERATOR INGRESS_GATEWAY CONTROL_PLANE PAAS_MEDIATION DBAAS_AGENT \
	MAAS_AGENT CORE_OPERATOR CONFIG_SERVER SITE_MANAGEMENT CORE_BOOTSTRAP CR_SYNCHRONIZER

$(eval $(call core_service,FACADE_OPERATOR,https://github.com/Netcracker/qubership-core-facade-operator.git))
# ingress-gateway is the exception to BASELINE_BRANCH: its repository has no branch for a release
# line (no lts/*), so main is its baseline whatever BASELINE_BRANCH is. TEST_BRANCH and an explicit
# INGRESS_GATEWAY_BRANCH still apply.
$(eval $(call core_service,INGRESS_GATEWAY,https://github.com/Netcracker/qubership-core-ingress-gateway.git,main))
$(eval $(call core_service,CONTROL_PLANE,https://github.com/Netcracker/qubership-core-control-plane.git))
$(eval $(call core_service,PAAS_MEDIATION,https://github.com/Netcracker/qubership-core-paas-mediation.git))
$(eval $(call core_service,DBAAS_AGENT,https://github.com/Netcracker/qubership-core-dbaas-agent.git))
$(eval $(call core_service,MAAS_AGENT,https://github.com/Netcracker/qubership-core-maas-agent.git))
$(eval $(call core_service,CORE_OPERATOR,https://github.com/Netcracker/qubership-core-core-operator.git))
$(eval $(call core_service,CONFIG_SERVER,https://github.com/Netcracker/qubership-core-config-server.git))
$(eval $(call core_service,SITE_MANAGEMENT,https://github.com/Netcracker/qubership-core-site-management.git))
# core-bootstrap and cr-synchronizer are images built from the repository this makefile lives in.
# Their repository is not cloned: only the image tag is resolved, from the branch of that repository.
$(eval $(call core_service,CORE_BOOTSTRAP,https://github.com/Netcracker/qubership-core-bootstrap.git))
$(eval $(call core_service,CR_SYNCHRONIZER,https://github.com/Netcracker/qubership-core-bootstrap.git))
$(eval $(call image_exists_or_baseline,CORE_BOOTSTRAP))
$(eval $(call image_exists_or_baseline,CR_SYNCHRONIZER))

$(eval $(call dependency_ref,DBAAS_REPO_BRANCH,https://github.com/Netcracker/qubership-dbaas.git))
$(eval $(call dependency_ref,MAAS_BRANCH,https://github.com/Netcracker/qubership-maas.git))
$(eval $(call dependency_ref,ISTIO_REPO_BRANCH,https://github.com/Netcracker/qubership-istio.git))
$(eval $(call dependency_ref,CORE_MESH_CONFIG_REPO_BRANCH,https://github.com/Netcracker/qubership-core-mesh-config.git))

# dbaas-aggregator and its hook image, not dbaas-agent; see run_dbaas_operation
ifeq ($(origin DBAAS_TAG),undefined)
DBAAS_TAG_SOURCE := derived from DBAAS_REPO_BRANCH
DBAAS_TAG := $(call ref_image_tag,https://github.com/Netcracker/qubership-dbaas.git,$(DBAAS_REPO_BRANCH))
else
DBAAS_TAG_SOURCE := explicit
endif

# The MaaS image tag is derived by the maas makefile; it is worked out here only to report it. A TAG
# set from outside reaches that makefile and wins there.
ifeq ($(origin TAG),undefined)
MAAS_IMAGE_TAG_SOURCE := derived by the maas makefile from MAAS_BRANCH
MAAS_IMAGE_TAG := $(call ref_image_tag,https://github.com/Netcracker/qubership-maas.git,$(MAAS_BRANCH))
else
MAAS_IMAGE_TAG_SOURCE := explicit (TAG)
MAAS_IMAGE_TAG := $(TAG)
endif

# =============================================================================
# RESOLVED BRANCHES AND TAGS REPORT
# =============================================================================
#
# resolved-refs prints every resolved branch and image tag as a Markdown table, with where the
# value came from: set from outside (explicit), TEST_BRANCH, BASELINE_BRANCH, a pin in
# dependencies.mk, or derived from a branch. Then it looks every image up in ghcr.io and prints a
# second table with the status of each: found, NOT FOUND, or not checked. install runs it first, so
# the report is at the top of the log, and it is printed in full before the install stops for an
# image that does not exist: there is no point in waiting for the cluster to fail pulling it.
# CHECK_IMAGES=false skips the lookup. When RESOLVED_REFS_FILE is set, the report is also appended
# to that file; a GitHub Actions step can point it at $GITHUB_STEP_SUMMARY to show it in the run
# summary.

# one row of the report: $(1) setting, $(2) value, $(3) where the value came from
define ref_row
echo "| \`$(1)\` | $(if $(2),\`$(2)\`,(not set)) | $(3) |"
endef

define resolved_refs_report
{ \
	echo ""; \
	echo "### Resolved branches and image tags"; \
	echo ""; \
	echo "| Setting | Value | Source |"; \
	echo "| --- | --- | --- |"; \
	$(call ref_row,TEST_BRANCH,$(TEST_BRANCH),$(if $(TEST_BRANCH),explicit,not set)); \
	$(call ref_row,BASELINE_BRANCH,$(BASELINE_BRANCH),$(if $(filter file default,$(origin BASELINE_BRANCH)),default (local.mk),explicit)); \
	$(call ref_row,M2M_AUTH_MODE,$(M2M_AUTH_MODE),$(if $(filter file default,$(origin M2M_AUTH_MODE)),default (local.mk),explicit)); \
	$(call ref_row,DBAAS_M2M_AUTH_MODE,$(DBAAS_M2M_AUTH_MODE),$(if $(filter file default,$(origin DBAAS_M2M_AUTH_MODE)),follows M2M_AUTH_MODE,explicit)); \
	$(call ref_row,MAAS_M2M_AUTH_MODE,$(MAAS_M2M_AUTH_MODE),$(if $(filter file default,$(origin MAAS_M2M_AUTH_MODE)),follows M2M_AUTH_MODE,explicit)); \
	$(call ref_row,INSTALL_DBAAS_AGENT,$(INSTALL_DBAAS_AGENT),$(if $(filter file default,$(origin INSTALL_DBAAS_AGENT)),default (off in the k8s mode),explicit)); \
	$(call ref_row,INSTALL_MAAS_AGENT,$(INSTALL_MAAS_AGENT),$(if $(filter file default,$(origin INSTALL_MAAS_AGENT)),default (off in the k8s mode),explicit)); \
	$(foreach s,$(CORE_SERVICES),$(call ref_row,$(s)_BRANCH,$($(s)_CLONE_BRANCH),$($(s)_BRANCH_SOURCE)); $(call ref_row,$(s)_TAG,$($(s)_TAG),$($(s)_TAG_SOURCE)); ) \
	$(call ref_row,DBAAS_REPO_BRANCH,$(DBAAS_REPO_BRANCH),$(DBAAS_REPO_BRANCH_SOURCE)); \
	$(call ref_row,DBAAS_TAG,$(DBAAS_TAG),$(DBAAS_TAG_SOURCE)); \
	$(call ref_row,MAAS_BRANCH,$(MAAS_BRANCH),$(MAAS_BRANCH_SOURCE)); \
	$(call ref_row,MaaS image tag,$(MAAS_IMAGE_TAG),$(MAAS_IMAGE_TAG_SOURCE)); \
	$(call ref_row,ISTIO_REPO_BRANCH,$(ISTIO_REPO_BRANCH),$(ISTIO_REPO_BRANCH_SOURCE)); \
	$(call ref_row,CORE_MESH_CONFIG_REPO_BRANCH,$(CORE_MESH_CONFIG_REPO_BRANCH),$(CORE_MESH_CONFIG_REPO_BRANCH_SOURCE)); \
	echo ""; \
}
endef

# A mode that is not one of the three would be passed on to the charts as it is.
$(foreach v,M2M_AUTH_MODE DBAAS_M2M_AUTH_MODE MAAS_M2M_AUTH_MODE,$(if $(filter legacy hybrid k8s,$($(v))),,$(error $(v) must be legacy, hybrid or k8s, got '$($(v))')))

.PHONY: resolved-refs check-images

# The images the install pulls, as repository:tag. DBaaS and MaaS install their own images, with
# the tags resolved above. The images of a component that is not installed (INSTALL_MAAS or
# INSTALL_DBAAS is not true) are left out of the lookup, so that a missing one cannot stop an install
# that does not use it; they are listed in SKIPPED_IMAGES and shown as not installed. The same goes
# for the images of the agents that INSTALL_DBAAS_AGENT and INSTALL_MAAS_AGENT leave out.
MAAS_IMAGES := $(MAAS_IMAGE_REPOSITORY):$(MAAS_IMAGE_TAG)
DBAAS_AGENT_IMAGE := $(DBAAS_AGENT_IMAGE_REPOSITORY):$(DBAAS_AGENT_TAG)
MAAS_AGENT_IMAGE := $(MAAS_AGENT_IMAGE_REPOSITORY):$(MAAS_AGENT_TAG)
DBAAS_IMAGES := $(DBAAS_IMAGE_REPOSITORY):$(DBAAS_TAG) $(DBAAS_VALIDATION_IMAGE_REPOSITORY):$(DBAAS_TAG)

RESOLVED_IMAGES := $(foreach s,$(filter-out CORE_BOOTSTRAP CR_SYNCHRONIZER DBAAS_AGENT MAAS_AGENT,$(CORE_SERVICES)),$($(s)_IMAGE_REPOSITORY):$($(s)_TAG)) \
	$(CORE_BOOTSTRAP_IMAGE) $(CR_SYNCHRONIZER_IMAGE) \
	$(if $(filter true,$(INSTALL_DBAAS_AGENT)),$(DBAAS_AGENT_IMAGE)) \
	$(if $(and $(filter true,$(INSTALL_MAAS)),$(filter true,$(INSTALL_MAAS_AGENT))),$(MAAS_AGENT_IMAGE)) \
	$(if $(filter true,$(INSTALL_DBAAS)),$(DBAAS_IMAGES)) \
	$(if $(filter true,$(INSTALL_MAAS)),$(MAAS_IMAGES))
SKIPPED_IMAGES := $(if $(filter true,$(INSTALL_DBAAS_AGENT)),,$(DBAAS_AGENT_IMAGE)) \
	$(if $(and $(filter true,$(INSTALL_MAAS)),$(filter true,$(INSTALL_MAAS_AGENT))),,$(MAAS_AGENT_IMAGE)) \
	$(if $(filter true,$(INSTALL_DBAAS)),,$(DBAAS_IMAGES)) \
	$(if $(filter true,$(INSTALL_MAAS)),,$(MAAS_IMAGES))

# Looks each image up in ghcr.io and prints a Markdown table: found, not found, or not checked for an
# image of another registry or a registry that cannot be reached. Only a 404 is "not found".
define image_check_report
{ \
	echo ""; \
	echo "### Images"; \
	echo ""; \
	echo "| Image | Status |"; \
	echo "| --- | --- |"; \
	for ref in $(RESOLVED_IMAGES); do \
		repo="$${ref%:*}"; tag="$${ref##*:}"; \
		case "$$repo" in \
			ghcr.io/*) ;; \
			*) echo "| \`$$ref\` | not checked, not in ghcr.io |"; continue ;; \
		esac; \
		path="$${repo#ghcr.io/}"; \
		token="$$(curl -fsS "https://ghcr.io/token?scope=repository:$$path:pull" 2>/dev/null | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"; \
		code="$$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $$token" \
			-H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json' \
			"https://ghcr.io/v2/$$path/manifests/$$tag" 2>/dev/null)"; \
		case "$$code" in \
			200) echo "| \`$$ref\` | found |" ;; \
			404) echo "| \`$$ref\` | **NOT FOUND** |" ;; \
			*) echo "| \`$$ref\` | not checked (HTTP $$code) |" ;; \
		esac; \
	done; \
	for ref in $(SKIPPED_IMAGES); do \
		echo "| \`$$ref\` | not installed |"; \
	done; \
	echo ""; \
}
endef

# The branches and tags report, then the image table. The whole report is printed, and appended to
# RESOLVED_REFS_FILE, before the target fails for an image that does not exist.
resolved-refs:
	@report="$$(mktemp)"; \
	$(call resolved_refs_report) > "$$report"; \
	$(if $(filter true,$(CHECK_IMAGES)),$(call image_check_report) >> "$$report"; ) \
	cat "$$report"; \
	$(if $(RESOLVED_REFS_FILE),cat "$$report" >> '$(RESOLVED_REFS_FILE)'; ) \
	if grep -q 'NOT FOUND' "$$report"; then \
		echo "$(RED)Error: these images do not exist in ghcr.io:$(NC)"; \
		grep 'NOT FOUND' "$$report" | cut -d'|' -f2; \
		echo "$(CYAN)Set the tag of the service, or TEST_BRANCH, BASELINE_BRANCH or the service's branch, to ones that have an image; CHECK_IMAGES=false skips this check.$(NC)"; \
		rm -f "$$report"; \
		exit 1; \
	fi; \
	rm -f "$$report"

# kept as the name of the check; it prints the same report
check-images: resolved-refs

# =============================================================================
# HELP
# =============================================================================

# Help target
help:
	@echo "$(BLUE)=== Cloud Core Local Development Makefile ===$(NC)"
	@echo ""
	@echo "$(CYAN)USAGE:$(NC)"
	@echo "  make [target] [CONFIG_FILE=local.mk] [DEPENDENCIES_FILE=dependencies.mk] [DEBUG=true]"
	@echo ""
	@echo "$(CYAN)CONFIGURATION VARIABLES:$(NC)"
	@echo "  CONFIG_FILE=local.mk           # Configuration file (default: local.mk)"
	@echo "  DEPENDENCIES_FILE=dependencies.mk  # Dependency tags/versions (default: dependencies.mk)"
	@echo "  DEBUG=true                     # Enable debug mode with --debug --dry-run"
	@echo ""
	@echo "$(CYAN)MAIN TARGETS:$(NC)"
	@echo "  install                 # Complete installation: validate prerequisites, deploy Cloud Core and dependencies, mesh connectivity smoke test"
	@echo "  deploy                  # Deploy Cloud Core and dependencies"
	@echo "  uninstall               # Complete uninstall, CRDs and namespaces cleanup"
	@echo ""
	@echo "$(CYAN)INSTALLATION TARGETS:$(NC)"
	@echo "  deploy-core             # Deploy Cloud Core bootstrap configuration and components"
	@echo "  deploy-dependencies     # Deploy metrics-server, monitoring, istio, consul, dbaas"
	@echo "  deploy-consul           # Deploy Consul (if INSTALL_CONSUL=true)"
	@echo "  deploy-monitoring       # Deploy monitoring operator (if INSTALL_MONITORING=true)"
	@echo "  deploy-metrics-server   # Deploy metrics server (if INSTALL_METRICS_SERVER=true)"
	@echo "  deploy-istio            # Deploy Istio via qubership-istio-distr (if INSTALL_ISTIO=true)"
	@echo "  deploy-core-mesh-config # Enable Istio mesh via core-mesh-config chart (if INSTALL_ISTIO=true)"
	@echo "  deploy-dbaas            # Deploy DBaaS (if INSTALL_DBAAS=true)"
	@echo "  deploy-maas             # Deploy MaaS (if INSTALL_MAAS=true)"
	@echo ""
	@echo "$(CYAN)VALIDATION TARGETS:$(NC)"
	@echo "  validate                # Validate CLI tools and CRDs"
	@echo "  validate-cli            # Check helm, git, kubectl availability"
	@echo "  validate-crd            # Check required CRDs"
	@echo "  check-consul-connectivity # Test Consul connectivity (if INSTALL_CONSUL=true)"
	@echo ""
	@echo "$(CYAN)TESTING TARGETS:$(NC)"
	@echo "  mesh-smoke-test         # Run mesh connectivity test"
	@echo "  cleanup-mesh-test       # Clean up mesh test resources"
	@echo ""
	@echo "$(CYAN)UNINSTALLATION TARGETS:$(NC)"
	@echo "  uninstall-core          # Uninstall Cloud Core bootstrap configuration and components"
	@echo "  uninstall-dependencies  # Uninstall dependencies"
	@echo "  uninstall-consul        # Uninstall Consul (if INSTALL_CONSUL=true)"
	@echo "  uninstall-monitoring    # Uninstall monitoring (if INSTALL_MONITORING=true)"
	@echo "  uninstall-metrics-server # Uninstall metrics server (if INSTALL_METRICS_SERVER=true)"
	@echo "  uninstall-istio         # Uninstall Istio (if INSTALL_ISTIO=true)"
	@echo "  uninstall-core-mesh-config # Disable Istio mesh, uninstall core-mesh-config chart (if INSTALL_ISTIO=true)"
	@echo "  uninstall-dbaas         # Uninstall DBaaS (if INSTALL_DBAAS=true)"
	@echo "  uninstall-maas          # Uninstall MaaS (if INSTALL_MAAS=true)"
	@echo "  cleanup-crds            # Remove all CRDs (if INSTALL_CRDS=true)"
	@echo "  cleanup-namespaces      # Remove all namespaces (if CREATE_NAMESPACE=true)"
	@echo ""
	@echo "$(CYAN)UTILITY TARGETS:$(NC)"
	@echo "  clone-repos             # Clone/update required repositories"
	@echo "  template-values-files   # Generate values files from config"
	@echo "  ensure-core-namespace   # Ensure core namespace exists"
	@echo "  reset-control-plane     # Reset control plane"
	@echo "  delete-control-plane-database # Delete control plane database"
	@echo "  restart-services        # Restart all services"
	@echo "  restart-public-gateway # Check public gateway config consistency"
	@echo "  restart-services-start # Print restart services start message"
	@echo "  restart-control-plane # Restart control plane"
	@echo "  check-public-gateway-config-no-mesh-test # Check public gateway config consistency"
	@echo ""
	@echo "$(CYAN)CONFIGURATION FILES:$(NC)"
	@echo "  local.mk                # Local development configuration (default)"
	@echo "  aws.mk                  # AWS environment configuration"
	@echo ""
	@echo "$(YELLOW)For more information, see README.md$(NC)"
	@echo ""

# =============================================================================
# INSTALLATION
# =============================================================================

install: resolved-refs validate clone-repos template-values-files deploy #mesh-smoke-test

deploy: deploy-dependencies deploy-core

deploy-dependencies: deploy-metrics-server deploy-monitoring deploy-consul deploy-dbaas deploy-maas deploy-istio
	@echo "$(GREEN)=== Cloud Core dependencies deployment completed$(NC)"
	@echo ""

deploy-core: ensure-core-namespace prepare-core-bootstrap deploy-cloud-core-components deploy-core-mesh-config
	@echo "$(GREEN)=== Cloud Core deployment completed$(NC)"
	@echo ""

#-------------------------------Validate prerequisites-------------------------------------------------------------

validate: validate-cli validate-crd

validate-cli:
	@echo "$(BLUE)=== CLI PREREQUISITE CHECKS ===$(NC)"
	@echo "$(CYAN)--- Checking if helm is available...$(NC)"
	@command -v helm >/dev/null 2>&1 || (echo "$(RED)Error: helm is not installed or not in PATH$(NC)" && echo "$(CYAN)Please install helm: https://helm.sh/docs/intro/install/$(NC)" && exit 1)
	echo "$(GREEN)--- helm4 found: $(shell helm version --short)$(NC)"
	@echo "$(CYAN)--- Checking if git is available...$(NC)"
	@command -v git >/dev/null 2>&1 || (echo "$(RED)Error: git is not installed or not in PATH$(NC)" && echo "$(CYAN)Please install git: https://git-scm.com/downloads$(NC)" && exit 1)
	@echo "$(GREEN)--- git found: $(shell git --version)$(NC)"
	@echo "$(CYAN)--- Checking if kubectl is available...$(NC)"
	@command -v kubectl >/dev/null 2>&1 || (echo "$(RED)Error: kubectl is not installed or not in PATH$(NC)" && echo "$(CYAN)Please install kubectl: https://kubernetes.io/docs/tasks/tools/$(NC)" && exit 1)
	@echo "$(GREEN)--- kubectl found: $(shell kubectl version --client)$(NC)"
	@echo "$(CYAN)--- Checking kubectl cluster connection...$(NC)"
	@kubectl cluster-info >/dev/null 2>&1 || (echo "$(RED)Error: kubectl cannot connect to cluster$(NC)" && echo "$(CYAN)Please check your kubeconfig and cluster status$(NC)" && exit 1)
	@echo "$(GREEN)--- kubectl cluster connection successful$(NC)"
	@echo ""

validate-crd:
	@echo "$(BLUE)=== CRD PREREQUISITE CHECKS ===$(NC)"
	@if [ "$(INSTALL_CRDS)" = "false" ]; then \
		echo "$(CYAN)--- Checking Facade Operator CRDs (INSTALL_CRDS=false)...$(NC)"; \
		$(call check_crd_existence,facadeservices.netcracker.com,facadeservices,kubectl apply -f $(REPOS_DIR)/qubership-core-facade-operator/helm-templates/facade-operator/new-crd/facade-service-crd-v1.yaml); \
		$(call check_crd_existence,gateways.core.netcracker.com,gateways,kubectl apply -f $(REPOS_DIR)/qubership-core-facade-operator/helm-templates/facade-operator/new-crd/gateway-crd-v1.yaml); \
		echo "$(CYAN)--- Checking Core Operator CRDs (INSTALL_CRDS=false)...$(NC)"; \
		$(call check_crd_existence,securities.core.netcracker.com,securities,kubectl apply -f $(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator/new-crd/security.yaml); \
		$(call check_crd_existence,meshes.core.netcracker.com,meshes,kubectl apply -f $(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator/new-crd/mesh.yaml); \
		$(call check_crd_existence,dbaases.core.netcracker.com,dbaases,kubectl apply -f $(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator/new-crd/dbaas.yaml); \
		$(call check_crd_existence,maases.core.netcracker.com,maases,kubectl apply -f $(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator/new-crd/maas.yaml); \
		$(call check_crd_existence,composites.core.netcracker.com,composites,kubectl apply -f $(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator/new-crd/composite.yaml); \
	else \
		echo "$(CYAN)--- Skipping additional CRD checks (INSTALL_CRDS=true, CRDs will be installed automatically)...$(NC)"; \
	fi
	@echo ""

#-------------------core namespace-----------------------------------------------------------------------------

# Core Namespace required for Consul connectivity check pod deployment
ensure-core-namespace:
	@if [ "$(CREATE_NAMESPACE)" = "true" ]; then \
		echo "$(GREEN)=== Ensuring Core namespace exists...$(NC)"; \
		echo ""; \
		$(call ensure_namespace_exists,$(CORE_NAMESPACE)); \
		echo "$(GREEN)=== Core namespace check completed$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Checking Core namespace existence (CREATE_NAMESPACE=false)...$(NC)"; \
		echo ""; \
		$(call check_namespace_existence,$(CORE_NAMESPACE)); \
		echo "$(GREEN)=== Core namespace check completed$(NC)"; \
		echo ""; \
	fi

# ---------------------- clone repos ---------------------------------------------

clone-repos:
	@echo "$(YELLOW)--- Cloning required repositories...$(NC)"
	mkdir -p $(REPOS_DIR)
	@$(call clone_or_update_repo,qubership-monitoring-operator,https://github.com/Netcracker/qubership-monitoring-operator.git,$(TEST_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-paas-mediation,https://github.com/Netcracker/qubership-core-paas-mediation.git,$(PAAS_MEDIATION_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-control-plane,https://github.com/Netcracker/qubership-core-control-plane.git,$(CONTROL_PLANE_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-ingress-gateway,https://github.com/Netcracker/qubership-core-ingress-gateway.git,$(INGRESS_GATEWAY_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-site-management,https://github.com/Netcracker/qubership-core-site-management.git,$(SITE_MANAGEMENT_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-facade-operator,https://github.com/Netcracker/qubership-core-facade-operator.git,$(FACADE_OPERATOR_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-dbaas-agent,https://github.com/Netcracker/qubership-core-dbaas-agent.git,$(DBAAS_AGENT_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-maas-agent,https://github.com/Netcracker/qubership-core-maas-agent.git,$(MAAS_AGENT_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-core-operator,https://github.com/Netcracker/qubership-core-core-operator.git,$(CORE_OPERATOR_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-core-config-server,https://github.com/Netcracker/qubership-core-config-server.git,$(CONFIG_SERVER_CLONE_BRANCH),$(BASELINE_BRANCH),branch-only)
	@$(call clone_or_update_repo,qubership-dbaas,https://github.com/Netcracker/qubership-dbaas.git,$(DBAAS_REPO_BRANCH),$(BASELINE_BRANCH))
	@echo ""

#---------------------------- values templating ---------------------------------------------

template-values-files:
	@envsubst < ./core-values.envsubst > ./core-values.yaml
	@echo "$(CYAN)=== core-values.yaml ===$(NC)"
	@cat ./core-values.yaml



# -------------------------- consul -----------------------

deploy-consul-old:
	@echo "$(GREEN)=== Deploying Consul...$(NC)"
	@echo ""
	helm upgrade --install --values ./consul-chart-3/values.yaml consul ./consul-chart-3/ --namespace consul $$(CREATE_NAMESPACE_OPTION)
	@echo "$(GREEN)=== Consul is deployed$(NC)"
	@echo ""

deploy-consul: deploy-consul-print-start validate-consul-namespace ensure-core-namespace clone-consul-repo change-consul-values wait-for-consul check-consul-connectivity
	@echo "$(GREEN)=== Consul setup completed...$(NC)"
	@echo ""

deploy-consul-print-start:
	@echo "$(GREEN)=== Starting Consul setup...$(NC)"
	@echo ""

validate-consul-namespace:
	@if [ "$(CREATE_NAMESPACE)" = "false" && "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(BLUE)=== CONSUL NAMESPACE EXISTENCE CHECKS ===$(NC)"; \
		$(call check_namespace_existence,$(CONSUL_NAMESPACE)); \
	else \
		echo "$(CYAN)--- Skipping Consul namespace checks (CREATE_NAMESPACE=true)...$(NC)"; \
	fi
	@echo ""

clone-consul-repo:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Cloning consul repository...$(NC)"; \
		$(call clone_or_update_repo,consul-k8s,https://github.com/hashicorp/consul-k8s,release/1.8.3); \
	else \
		echo "$(YELLOW)--- Skipping Consul repository cloning (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

change-consul-values:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Updating Consul Helm chart security context...$(NC)"; \
		if [ -f $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl ]; then \
			if command -v sed >/dev/null 2>&1; then \
				if sed --version 2>/dev/null | grep -q "GNU sed"; then \
					sed -i 's/runAsNonRoot: true/runAsNonRoot: false/g' $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl; \
					sed -i 's/runAsUser: 100/runAsUser: 0/g' $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl; \
				else \
					sed -i '' 's/runAsNonRoot: true/runAsNonRoot: false/g' $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl; \
					sed -i '' 's/runAsUser: 100/runAsUser: 0/g' $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl; \
				fi; \
			else \
				echo "Warning: sed not found, using alternative method"; \
				cp $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl.bak; \
				cat $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl.bak | sed 's/runAsNonRoot: true/runAsNonRoot: false/g' | sed 's/runAsUser: 100/runAsUser: 0/g' > $(REPOS_DIR)/consul-k8s/charts/consul/templates/_helpers.tpl; \
			fi; \
		echo "Security context updated in _helpers.tpl"; \
	else \
		echo "Error: _helpers.tpl not found"; \
		exit 1; \
		fi; \
	else \
		echo "$(YELLOW)--- Skipping Consul values update (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

install-consul:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Installing consul...$(NC)"; \
		helm upgrade --install --namespace $(CONSUL_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
			consul $(REPOS_DIR)/consul-k8s/charts/consul/ \
			-f $(REPOS_DIR)/consul-k8s/charts/consul/values.yaml \
			--set server.securityContext.runAsNonRoot=false \
			--set server.securityContext.runAsUser=0 \
			--set connectInject.enabled=false \
			$(CONSUL_ACLS_OPTION) \
			--set server.storage=1Gi \
			--set global.imageK8S='docker.io/hashicorp/consul-k8s-control-plane:latest' \
			--set global.image='docker.io/hashicorp/consul:latest'; \
	else \
		echo "$(YELLOW)--- Skipping Consul installation (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

wait-for-consul: install-consul
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Waiting for Consul pods to be ready (timeout: 5 minutes)...$(NC)"; \
		TIMEOUT=300; \
		START_TIME=$$(date +%s); \
		while true; do \
			CURRENT_TIME=$$(date +%s); \
			ELAPSED_TIME=$$((CURRENT_TIME - START_TIME)); \
			if [ $$ELAPSED_TIME -ge $$TIMEOUT ]; then \
				echo "Timeout reached after 5 minutes. Consul pods are not ready."; \
				exit 1; \
			fi; \
			RUNNING_CONSUL_PODS=$$(kubectl -n $(CONSUL_NAMESPACE) get pods --selector=app=consul --field-selector=status.phase=Running -o name | wc -l); \
			if [ "$$RUNNING_CONSUL_PODS" -ge 1 ]; then \
				echo "Consul pods are running ($$RUNNING_CONSUL_PODS)!"; \
				break; \
			fi; \
			echo "Waiting for Consul pods to be ready... ($$RUNNING_CONSUL_PODS) - $$((TIMEOUT - ELAPSED_TIME))s remaining"; \
			sleep 10; \
		done; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Consul wait (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

# Include Consul targets
include test/consul-connectivity-test/consul-connectivity-test.mk

# ----------------- monitoring --------------------------

deploy-monitoring:
	@if [ "$(INSTALL_MONITORING)" = "true" ]; then \
		echo "$(GREEN)=== Deploying Monitoring...$(NC)"; \
		echo ""; \
		if [ "$(CREATE_NAMESPACE)" = "false" ]; then \
			$(call check_namespace_existence,$(MONITORING_NAMESPACE)); \
			echo ""; \
		fi; \
		helm upgrade --install --namespace $(MONITORING_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
			monitoring-operator $(REPOS_DIR)/qubership-monitoring-operator/charts/qubership-monitoring-operator \
			-f $(REPOS_DIR)/qubership-monitoring-operator/charts/qubership-monitoring-operator/values.yaml; \
		echo "$(GREEN)=== Monitoring is deployed$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Monitoring deployment (MONITORING_ENABLED=false)...$(NC)"; \
		echo ""; \
	fi

# ------------------- metrics-server -------------------------------

deploy-metrics-server:
	@if [ "$(INSTALL_METRICS_SERVER)" = "true" ]; then \
		echo "$(GREEN)=== Deploying Metrics Server...$(NC)"; \
		echo ""; \
		kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml; \
		kubectl patch deployment metrics-server -n kube-system --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]'; \
		echo "$(GREEN)=== Metrics Server is deployed$(NC)"; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Metrics Server deployment (INSTALL_METRICS_SERVER=false)...$(NC)"; \
		echo ""; \
	fi

# --------------------- bootstrap --------------------

prepare-core-bootstrap: prepare-core-bootstrap-print-start deploy-cloud-core-configuration
	@echo "$(GREEN)=== Bootstrap preparation completed$(NC)"
	@echo ""

prepare-core-bootstrap-print-start:
	@echo "$(GREEN)=== Starting bootstrap preparation...$(NC)"
	@echo ""

deploy-cloud-core-configuration:
	@echo "$(YELLOW)--- Deploying cloud-core-configuration$(NC)"
	@$(call update_helm_dependencies,../cloud-core-configuration/helm-charts,)
	helm upgrade --install -n $(CORE_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		cloud-core-app-chart ../cloud-core-configuration/helm-charts \
		-f ./core-values.yaml \
		--set CONSUL_ENABLED=$(CORE_CONFIG_CONSUL_ENABLED) \
		--set MAAS_ENABLED=$(CORE_CONFIG_MAAS_ENABLED) \
		--set MAAS_INTERNAL_ADDRESS=$(CORE_CONFIG_MAAS_INTERNAL_ADDRESS) \
		--set NAMESPACE=$(CORE_NAMESPACE) \
		--set CORE_BOOTSTRAP_IMAGE=$(CORE_BOOTSTRAP_IMAGE) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

# ------------------ cloud core ---------------------------

deploy-cloud-core-components: deploy-cloud-core-print-start deploy-facade-operator deploy-ingress-gateway deploy-control-plane deploy-paas-mediation deploy-dbaas-agent deploy-maas-agent deploy-core-operator deploy-config-server deploy-site-management
	@echo "$(GREEN)=== Cloud Core components deployment completed$(NC)"
	@echo ""

deploy-cloud-core-print-start:
	@echo "$(GREEN)=== Starting Cloud Core installation...$(NC)"
	@echo ""

apply-facade-operator-crds:
	@if [ "$(INSTALL_CRDS)" = "true" ]; then \
		echo "$(YELLOW)--- Create Facade Operator CRDs...$(NC)"; \
		echo "Creating facadeservices.netcracker.com, gateways.core.netcracker.com CRDs..."; \
		kubectl apply -f $(REPOS_DIR)/qubership-core-facade-operator/helm-templates/facade-operator/new-crd; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Facade Operator CRDs installation (INSTALL_CRDS=false)...$(NC)"; \
		echo ""; \
	fi

deploy-facade-operator: apply-facade-operator-crds
	@echo "$(YELLOW)--- Setting up Facade Operator...$(NC)"
	@$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-facade-operator/helm-templates/facade-operator,)
	helm upgrade --install facade-operator $(REPOS_DIR)/qubership-core-facade-operator/helm-templates/facade-operator -n $(CORE_NAMESPACE) \
		-f $(REPOS_DIR)/qubership-core-facade-operator/helm-templates/facade-operator/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set IMAGE_REPOSITORY=$(FACADE_OPERATOR_IMAGE_REPOSITORY) \
		--set TAG=$(FACADE_OPERATOR_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

# SERVICE_MESH_TYPE is set to Istio when Istio installed, as it contains contract gateway services, that must point to Istio gateways in this case
# that is why SERVICE_MESH_TYPE from core-values.yaml is overriden here 
deploy-ingress-gateway: 
	@echo "$(YELLOW)--- Setting up Ingress Gateway...$(NC)"
	@$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-ingress-gateway/helm-templates/frontend-gateway,)
	helm upgrade --install ingress-gateway $(REPOS_DIR)/qubership-core-ingress-gateway/helm-templates/frontend-gateway -n $(CORE_NAMESPACE) \
		-f $(REPOS_DIR)/qubership-core-ingress-gateway/helm-templates/frontend-gateway/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set CLOUD_PUBLIC_HOST=$(INGRESS_GATEWAY_CLOUD_PUBLIC_HOST) \
		--set CLOUD_PRIVATE_HOST=$(INGRESS_GATEWAY_CLOUD_PRIVATE_HOST) \
		--set IMAGE_REPOSITORY=$(INGRESS_GATEWAY_IMAGE_REPOSITORY) \
		--set TAG=$(INGRESS_GATEWAY_TAG) \
		--set SERVICE_MESH_TYPE=$(if $(filter true,$(INSTALL_ISTIO)),Istio,Core) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-site-management: 
	@echo "$(YELLOW)--- Setting up Site Management...$(NC)"
	@$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-site-management/helm-templates/site-management,)
	helm upgrade --install site-management $(REPOS_DIR)/qubership-core-site-management/helm-templates/site-management -n $(CORE_NAMESPACE) \
		-f $(REPOS_DIR)/qubership-core-site-management/helm-templates/site-management/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set IMAGE_REPOSITORY=$(SITE_MANAGEMENT_IMAGE_REPOSITORY) \
		--set TAG=$(SITE_MANAGEMENT_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-paas-mediation: 
	@echo "$(YELLOW)--- Setting up PaaS Mediation...$(NC)"
	@$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-paas-mediation/helm-templates/paas-mediation,)
	helm upgrade --install -n $(CORE_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		paas-mediation $(REPOS_DIR)/qubership-core-paas-mediation/helm-templates/paas-mediation \
		-f $(REPOS_DIR)/qubership-core-paas-mediation/helm-templates/paas-mediation/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set IMAGE_REPOSITORY=$(PAAS_MEDIATION_IMAGE_REPOSITORY) \
		--set TAG=$(PAAS_MEDIATION_TAG) \
		--set CONSUL_ENABLED=false \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-control-plane: 
	@echo "$(YELLOW)--- Setting up Control Plane...$(NC)"
	@$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-control-plane/helm-templates/control-plane,)
	helm upgrade --install -n $(CORE_NAMESPACE) $(CREATE_NAMESPACE_OPTION) \
		control-plane $(REPOS_DIR)/qubership-core-control-plane/helm-templates/control-plane \
		-f $(REPOS_DIR)/qubership-core-control-plane/helm-templates/control-plane/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set IMAGE_REPOSITORY=$(CONTROL_PLANE_IMAGE_REPOSITORY) \
		--set TAG=$(CONTROL_PLANE_TAG) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-dbaas-agent:
	@if [ "$(INSTALL_DBAAS_AGENT)" = "true" ]; then \
		echo "$(YELLOW)--- Setting up DBaaS agent...$(NC)"; \
		$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-dbaas-agent/helm-templates/dbaas-agent,); \
		helm upgrade --install dbaas-agent $(REPOS_DIR)/qubership-core-dbaas-agent/helm-templates/dbaas-agent -n $(CORE_NAMESPACE) \
			-f $(REPOS_DIR)/qubership-core-dbaas-agent/helm-templates/dbaas-agent/resource-profiles/dev.yaml \
			-f ./core-values.yaml \
			--set IMAGE_REPOSITORY=$(DBAAS_AGENT_IMAGE_REPOSITORY) \
			--set TAG=$(DBAAS_AGENT_TAG) \
			$(HELM_DEBUG_FLAGS); \
		echo ""; \
	else \
		echo "$(CYAN)--- Skipping DBaaS agent installation (INSTALL_DBAAS_AGENT=false)...$(NC)"; \
		echo ""; \
	fi
	@echo ""

deploy-maas-agent:
	@if [ "$(INSTALL_MAAS)" = "true" ] && [ "$(INSTALL_MAAS_AGENT)" = "true" ]; then \
		echo "$(YELLOW)--- Setting up MaaS agent...$(NC)"; \
		$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-maas-agent/helm-templates/maas-agent,); \
		helm upgrade --install maas-agent $(REPOS_DIR)/qubership-core-maas-agent/helm-templates/maas-agent -n $(CORE_NAMESPACE) \
			-f $(REPOS_DIR)/qubership-core-maas-agent/helm-templates/maas-agent/resource-profiles/dev.yaml \
			-f ./core-values.yaml \
			--set IMAGE_REPOSITORY=$(MAAS_AGENT_IMAGE_REPOSITORY) \
			--set TAG=$(MAAS_AGENT_TAG) \
			--set MAAS_ENABLED=$(CORE_CONFIG_MAAS_ENABLED) \
			--set MAAS_INTERNAL_ADDRESS=$(CORE_CONFIG_MAAS_INTERNAL_ADDRESS) \
			$(HELM_DEBUG_FLAGS); \
		echo ""; \
	else \
		echo "$(CYAN)--- Skipping MaaS agent installation (INSTALL_MAAS=$(INSTALL_MAAS), INSTALL_MAAS_AGENT=$(INSTALL_MAAS_AGENT))...$(NC)"; \
		echo ""; \
	fi
	@echo ""

apply-core-operator-crds:
	@if [ "$(INSTALL_CRDS)" = "true" ]; then \
		echo "$(YELLOW)--- Applying Core Operator CRDs...$(NC)"; \
		kubectl apply -f $(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator/new-crd; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Core Operator CRDs installation (INSTALL_CRDS=false)...$(NC)"; \
		echo ""; \
	fi

# When Cloud Core's DBaaS service name is faked (CORE_DBAAS_SERVICE_NAME), core-operator must not send
# the composite structure to DBaaS: the request goes through dbaas-agent to that name, and the Composite
# never becomes ready. MaaS, when enabled, stays a receiver.
deploy-core-operator: apply-core-operator-crds
	@echo "$(YELLOW)--- Setting up Core Operator...$(NC)"
	@$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator,)
	helm upgrade --install core-operator $(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator -n $(CORE_NAMESPACE) \
		-f $(REPOS_DIR)/qubership-core-core-operator/helm-templates/core-operator/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set IMAGE_REPOSITORY=$(CORE_OPERATOR_IMAGE_REPOSITORY) \
		--set TAG=$(CORE_OPERATOR_TAG) \
		$(if $(filter-out $(DBAAS_SERVICE_NAME),$(CORE_DBAAS_SERVICE_NAME)),--set COMPOSITE_STRUCTURE_XAAS_RECEIVERS=$(if $(filter true,$(CORE_CONFIG_MAAS_ENABLED)),maas,none),) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

deploy-config-server: 
	@echo "$(YELLOW)--- Setting up Config Server...$(NC)"
	@$(call update_helm_dependencies,$(REPOS_DIR)/qubership-core-config-server/helm-templates/config-server,)
	helm upgrade --install config-server $(REPOS_DIR)/qubership-core-config-server/helm-templates/config-server -n $(CORE_NAMESPACE) \
		-f $(REPOS_DIR)/qubership-core-config-server/helm-templates/config-server/resource-profiles/dev.yaml \
		-f ./core-values.yaml \
		--set IMAGE_REPOSITORY=$(CONFIG_SERVER_IMAGE_REPOSITORY) \
		--set TAG=$(CONFIG_SERVER_TAG) \
		--set CONSUL_ENABLED=$(CONFIG_SERVER_CONSUL_ENABLED) \
		$(HELM_DEBUG_FLAGS)
	@echo ""

# =============================================================================
# UNINSTALL
# =============================================================================

uninstall: uninstall-istio uninstall-maas uninstall-core uninstall-dependencies cleanup-mesh-test cleanup-crds cleanup-namespaces
	@echo "$(GREEN)=== Complete cleanup finished$(NC)"
	@echo ""

uninstall-core: uninstall-core-mesh-config delete-control-plane-database uninstall-cloud-core-components uninstall-cloud-core-configuration
	@echo "$(GREEN)=== Core uninstallation completed$(NC)"
	@echo ""

uninstall-cloud-core-components: uninstall-site-management uninstall-config-server uninstall-core-operator uninstall-dbaas-agent uninstall-maas-agent uninstall-paas-mediation uninstall-control-plane uninstall-ingress-gateway uninstall-facade-operator
	@echo "$(GREEN)=== Cloud Core components uninstallation completed$(NC)"
	@echo ""

uninstall-dependencies: uninstall-dbaas uninstall-consul uninstall-monitoring uninstall-metrics-server
	@echo "$(GREEN)=== Dependencies uninstallation completed$(NC)"
	@echo ""

# -----------------------------------------Cloud Core components uninstall targets-------------------------------------

uninstall-cloud-core-configuration: uninstall-control-plane-db-credentials
	@echo "$(YELLOW)--- Uninstalling cloud-core-app-chart...$(NC)"
	helm uninstall cloud-core-app-chart -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""
	@echo ""

uninstall-control-plane-db-credentials:
	@echo "$(YELLOW)--- Uninstalling control-plane-db-credentials...$(NC)"
	kubectl delete secret control-plane-db-credentials -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-facade-operator:
	@echo "$(YELLOW)--- Uninstalling facade-operator...$(NC)"
	helm uninstall facade-operator -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo "$(YELLOW)--- Removing leftover facade-operator deployments...$(NC)"
	kubectl delete deployment -n $(CORE_NAMESPACE) -l app.kubernetes.io/name=facade-operator --ignore-not-found=true || true
	@echo ""

uninstall-ingress-gateway:
	@echo "$(YELLOW)--- Uninstalling ingress-gateway...$(NC)"
	helm uninstall ingress-gateway -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo "$(YELLOW)--- Removing leftover FacadeService helm hook (core-egress-gateway)...$(NC)"
	kubectl delete facadeservice.v1alpha.netcracker.com core-egress-gateway -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-site-management:
	@echo "$(YELLOW)--- Uninstalling site-management...$(NC)"
	helm uninstall site-management -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-paas-mediation:
	@echo "$(YELLOW)--- Uninstalling paas-mediation...$(NC)"
	helm uninstall paas-mediation -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-control-plane: uninstall-control-plane-db-credentials
	@echo "$(YELLOW)--- Uninstalling control-plane...$(NC)"
	helm uninstall control-plane -n $(CORE_NAMESPACE) --ignore-not-found=true || true

uninstall-dbaas-agent:
	@echo "$(YELLOW)--- Uninstalling dbaas-agent...$(NC)"
	helm uninstall dbaas-agent -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-maas-agent:
	@echo "$(YELLOW)--- Uninstalling maas-agent...$(NC)"
	helm uninstall maas-agent -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-core-operator:
	@echo "$(YELLOW)--- Uninstalling core-operator...$(NC)"
	helm uninstall core-operator -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

uninstall-config-server:
	@echo "$(YELLOW)--- Uninstalling config-server...$(NC)"
	helm uninstall config-server -n $(CORE_NAMESPACE) --ignore-not-found=true || true
	@echo ""

# -----------------------------------------Dependencies uninstall targets-------------------------------------

uninstall-consul:
	@if [ "$(INSTALL_CONSUL)" = "true" ]; then \
		echo "$(YELLOW)--- Uninstalling consul...$(NC)"; \
		helm uninstall consul -n $(CONSUL_NAMESPACE) --ignore-not-found=true || true; \
		kubectl delete pvc -n $(CONSUL_NAMESPACE) --all --ignore-not-found=true || true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Consul uninstallation (INSTALL_CONSUL=false)...$(NC)"; \
		echo ""; \
	fi

uninstall-monitoring:
	@if [ "$(INSTALL_MONITORING)" = "true" ]; then \
		echo "$(YELLOW)--- Uninstalling monitoring-operator...$(NC)"; \
		helm uninstall monitoring-operator -n $(MONITORING_NAMESPACE) --ignore-not-found=true || true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping Monitoring uninstallation (INSTALL_MONITORING=false)...$(NC)"; \
		echo ""; \
	fi

uninstall-metrics-server:
	@if [ "$(INSTALL_METRICS_SERVER)" = "true" ]; then \
		echo "$(YELLOW)--- Cleaning up metrics-server...$(NC)"; \
		kubectl delete -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml --ignore-not-found=true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping metrics-server cleanup (INSTALL_METRICS_SERVER=false)...$(NC)"; \
		echo ""; \
	fi

cleanup-crds:
	@echo "$(YELLOW)--- Cleaning up CRDs...$(NC)"
	@if [ "$(INSTALL_CRDS)" = "true" ]; then \
		echo "Removing Facade Operator CRDs..."; \
		kubectl delete crd facadeservices.netcracker.com --ignore-not-found=true; \
		kubectl delete crd gateways.core.netcracker.com --ignore-not-found=true; \
		echo "Removing Core Operator CRDs..."; \
		kubectl delete crd securities.core.netcracker.com --ignore-not-found=true; \
		kubectl delete crd meshes.core.netcracker.com --ignore-not-found=true; \
		kubectl delete crd dbaases.core.netcracker.com --ignore-not-found=true; \
		kubectl delete crd maases.core.netcracker.com --ignore-not-found=true; \
		kubectl delete crd composites.core.netcracker.com --ignore-not-found=true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping CRD cleanup (INSTALL_CRDS=false)...$(NC)"; \
		echo ""; \
	fi

cleanup-namespaces:
	@echo "$(YELLOW)--- Cleaning up namespaces...$(NC)"
	@if [ "$(CREATE_NAMESPACE)" = "true" ]; then \
		echo "Removing namespaces..."; \
		if [ "$(INSTALL_CONSUL)" = "true" ]; then \
			kubectl delete namespace $(CONSUL_NAMESPACE) --ignore-not-found=true; \
		fi; \
		if [ "$(INSTALL_MONITORING)" = "true" ]; then \
			kubectl delete namespace $(MONITORING_NAMESPACE) --ignore-not-found=true; \
		fi; \
		kubectl delete namespace $(CORE_NAMESPACE) --ignore-not-found=true; \
		echo ""; \
	else \
		echo "$(YELLOW)--- Skipping namespace cleanup (CREATE_NAMESPACE=false)...$(NC)"; \
		echo ""; \
	fi

# =============================================================================
# MESH SMOKE TEST
# =============================================================================

mesh-smoke-test:
	@echo "$(GREEN)=== Starting Cloud Core Mesh Smoke Test...$(NC)"
	@'$(MAKE)' -C test/mesh-test mesh-smoke-test NAMESPACE=$(CORE_NAMESPACE)
	@echo ""

cleanup-mesh-test:
	@echo "$(YELLOW)--- Cleaning up mesh test resources...$(NC)"
	@'$(MAKE)' -C test/mesh-test cleanup-mesh-test NAMESPACE=$(CORE_NAMESPACE)
	@echo ""

# =============================================================================
# DBAAS
# =============================================================================

deploy-dbaas:
	@if [ "$(INSTALL_DBAAS)" = "true" ]; then \
		echo "$(GREEN)=== Starting DBaaS Installation...$(NC)"; \
		echo ""; \
		if [ -d "dbaas" ]; then \
			echo "$(CYAN)--- Running DBaaS installation from dbaas directory...$(NC)"; \
			$(call run_dbaas_operation,install); \
			echo ""; \
		else \
			echo "$(RED)Error: dbaas directory not found$(NC)"; \
			echo "$(CYAN)Please ensure the dbaas directory exists in the current location$(NC)"; \
			exit 1; \
		fi; \
	else \
		echo "$(CYAN)--- Skipping DBaaS installation (INSTALL_DBAAS=false)...$(NC)"; \
		echo "$(GREEN)=== Running DBaaS smoke test on existing installation...$(NC)"; \
		echo ""; \
		if [ -d "dbaas" ]; then \
			echo "$(CYAN)--- Running DBaaS smoke test from dbaas directory...$(NC)"; \
			$(call run_dbaas_operation,smoke-test); \
		else \
			echo "$(RED)Error: dbaas directory not found$(NC)"; \
			echo "$(CYAN)Please ensure the dbaas directory exists in the current location$(NC)"; \
			exit 1; \
		fi; \
	fi

uninstall-dbaas:
	@if [ "$(INSTALL_DBAAS)" = "true" ]; then \
		echo "$(YELLOW)--- Uninstalling DBaaS components...$(NC)"; \
		if [ -d "dbaas" ]; then \
			echo "$(YELLOW)--- Running DBaaS uninstallation from dbaas directory...$(NC)"; \
			$(call run_dbaas_operation,uninstall); \
			echo ""; \
		else \
			echo "$(RED)Error: dbaas directory not found$(NC)"; \
			echo "$(CYAN)Please ensure the dbaas directory exists in the current location$(NC)"; \
			exit 1; \
		fi; \
	else \
		echo "$(YELLOW)--- Skipping DBaaS uninstallation (INSTALL_DBAAS=false)...$(NC)"; \
		echo ""; \
	fi

# =============================================================================
# MAAS
# =============================================================================

deploy-maas:
	@echo "$(GREEN)=== Starting MaaS installation...$(NC)"
	@echo ""
	@if [ "$(INSTALL_MAAS)" = "true" ]; then \
		$(call run_maas_operation,install); \
	else \
		echo "$(CYAN)--- Skipping MaaS installation (INSTALL_MAAS=false)...$(NC)"; \
		echo ""; \
	fi
	@echo ""

uninstall-maas:
	@echo "$(GREEN)=== Starting MaaS uninstallation...$(NC)"
	@echo ""
	@if [ "$(INSTALL_MAAS)" = "true" ]; then \
		$(call run_maas_operation,uninstall); \
	else \
		echo "$(CYAN)--- Skipping MaaS uninstallation (INSTALL_MAAS=false)...$(NC)"; \
		echo ""; \
	fi
	@echo ""

# =============================================================================
# ISTIO
# =============================================================================
deploy-istio:
	@echo "$(GREEN)=== Starting Istio installation...$(NC)"
	@echo ""
	@if [ "$(INSTALL_ISTIO)" = "true" ]; then \
		$(call run_istio_operation,install); \
	else \
		echo "$(CYAN)--- Skipping Istio installation (INSTALL_ISTIO=false)...$(NC)"; \
		echo ""; \
	fi
	@echo ""

uninstall-istio:
	@echo "$(GREEN)=== Starting Istio uninstallation...$(NC)"
	@echo ""
	@if [ "$(INSTALL_ISTIO)" = "true" ]; then \
		$(call run_istio_operation,uninstall); \
	else \
		echo "$(CYAN)--- Skipping Istio uninstallation (INSTALL_ISTIO=false)...$(NC)"; \
		echo ""; \
	fi
	@echo ""

# =============================================================================
# CORE MESH CONFIG
# =============================================================================
deploy-core-mesh-config:
	@echo "$(GREEN)=== Starting core mesh config installation...$(NC)"
	@echo ""
	@$(call run_core_mesh_config_operation,enable);
	@echo ""

uninstall-core-mesh-config:
	@echo "$(GREEN)=== Starting core mesh config uninstallation...$(NC)"
	@echo ""
	@$(call run_core_mesh_config_operation,disable);
	@echo ""


# =============================================================================
# Reset control plane targets
# =============================================================================
include utils/reset-control-plane.mk

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

# Clone a repo into $(REPOS_DIR), replacing any existing directory. $(3) may be a branch, tag or
# commit sha.
# Usage: $(call clone_or_update_repo,<local_name>,<git_url>,<ref>,<baseline_branch>[,<branch_only>])
#   $(1) Directory name under $(REPOS_DIR) for the clone.
#   $(2) Remote repository URL.
#   $(3) Optional ref: a branch, tag or commit sha. Used when non-empty and it resolves on
#        remote; otherwise falls back to $(4).
#   $(4) Baseline branch (main, or a release line such as lts/26.3): cloned when present on
#        remote and $(3) does not resolve; otherwise falls back to the remote default. Defaults to
#        main when left empty, so existing callers that only pass $(1)-$(3) keep working.
#   $(5) Optional: when non-empty, $(3) is looked up as a branch only; a tag or a commit sha of
#        that name does not count. Used for TEST_BRANCH.
define clone_or_update_repo
	baseline="$(if $(4),$(4),main)"; \
	if [ -d "$(REPOS_DIR)/$(1)" ]; then \
		echo "Cleaning $(1)..."; \
		rm -rf "$(REPOS_DIR)/$(1)"; \
	fi; \
	echo "Cloning $(1)..."; \
	if [ -n "$(3)" ] && [ -n "$$(git ls-remote $(if $(5),--heads) $(2) $(if $(5),refs/heads/)$(3) 2>/dev/null)" ]; then \
		git clone --depth 1 -b $(3) $(2) $(REPOS_DIR)/$(1); \
	elif [ -z "$(5)" ] && echo "$(3)" | grep -qE '^[0-9a-fA-F]{7,40}$$'; then \
		echo "$(3) is not a branch or tag on remote, fetching it as a commit..."; \
		mkdir -p $(REPOS_DIR)/$(1); \
		if (cd $(REPOS_DIR)/$(1) && git init -q && git remote add origin $(2) && git fetch --depth 1 origin $(3) && git checkout -q FETCH_HEAD); then \
			:; \
		else \
			echo "Commit $(3) could not be fetched, cloning $$baseline instead..."; \
			rm -rf $(REPOS_DIR)/$(1); \
			git clone --depth 1 -b "$$baseline" $(2) $(REPOS_DIR)/$(1); \
		fi; \
	elif [ -n "$$(git ls-remote $(2) "$$baseline" 2>/dev/null)" ]; then \
		if [ -n "$(3)" ]; then \
			echo "$(if $(5),Branch,Branch, tag or commit) $(3) not found, cloning $$baseline..."; \
		fi; \
		git clone --depth 1 -b "$$baseline" $(2) $(REPOS_DIR)/$(1); \
	else \
		echo "Branch $$baseline not found on remote, cloning default branch..."; \
		git clone --depth 1 $(2) $(REPOS_DIR)/$(1); \
	fi
endef

# Helper function to check namespace existence
define check_namespace_existence
	set -e; \
	echo "$(CYAN)--- Checking if namespace $(1) exists...$(NC)"; \
	kubectl get namespace $(1) >/dev/null 2>&1 || (echo "$(RED)Error: Namespace $(1) not found$(NC)" && echo "$(CYAN)Please create the namespace first:$(NC)" && echo "$(CYAN)kubectl create namespace $(1)$(NC)" && exit 1); \
	echo "$(GREEN)--- Namespace $(1) found$(NC)"
endef

# Helper function to check and create namespace if it doesn't exist
define ensure_namespace_exists
	echo "$(CYAN)--- Checking if namespace $(1) exists...$(NC)"; \
	if kubectl get namespace $(1) >/dev/null 2>&1; then \
		echo "$(GREEN)--- Namespace $(1) already exists$(NC)"; \
	else \
		echo "$(YELLOW)--- Namespace $(1) not found, creating it...$(NC)"; \
		kubectl create namespace $(1); \
		echo "$(GREEN)--- Namespace $(1) created successfully$(NC)"; \
	fi
endef

# Helper function to check CRD existence
define check_crd_existence
	set -e; \
	echo "$(CYAN)--- Checking if CRD $(1) exists...$(NC)"; \
	kubectl api-resources | grep -q $(2) || (echo "$(RED)Error: CRD $(1) not found$(NC)" && echo "$(CYAN)Please install the CRD first:$(NC)" && echo "$(CYAN)$(3)$(NC)" && exit 1); \
	echo "$(GREEN)--- CRD $(1) found$(NC)"
endef

# Helper function to update Helm dependencies.
# With CORETPL_CHART_DIR set, the chart gets coretpl packaged from that directory instead, so an
# unpublished coretpl can be tested. The update is skipped then: coretpl is the only dependency of
# the Cloud Core charts, and the update would fetch the published version again.
define update_helm_dependencies
	if [ -n "$(CORETPL_CHART_DIR)" ]; then \
		echo "$(CYAN)--- Packaging coretpl from $(CORETPL_CHART_DIR) into $(1)/charts...$(NC)"; \
		mkdir -p $(1)/charts && rm -f $(1)/charts/coretpl-*.tgz; \
		if helm package $(abspath $(CORETPL_CHART_DIR)) --destination $(1)/charts; then \
			echo "$(GREEN)--- coretpl packaged successfully$(NC)"; \
		else \
			echo "$(RED)Error: Failed to package coretpl from $(CORETPL_CHART_DIR)$(NC)"; \
			exit 1; \
		fi; \
	elif [ -n "$(2)" ]; then \
		echo "$(CYAN)--- Updating Helm dependencies in $(1) with config $(2)...$(NC)"; \
		if (cd $(1) && helm dependency update --repository-config $(2)); then \
			echo "$(GREEN)--- Helm dependencies updated successfully$(NC)"; \
		else \
			echo "$(RED)Error: Failed to update Helm dependencies$(NC)"; \
			exit 1; \
		fi; \
	else \
		echo "$(CYAN)--- Updating Helm dependencies in $(1) with config ./core-bootstrap-helm-repo.yaml...$(NC)"; \
		if (cd $(1) && helm dependency update --repository-config ./core-bootstrap-helm-repo.yaml); then \
			echo "$(GREEN)--- Helm dependencies updated successfully$(NC)"; \
		else \
			echo "$(RED)Error: Failed to update Helm dependencies$(NC)"; \
			exit 1; \
		fi; \
	fi
endef

# Helper function to run DBaaS operations
# REPOS_DIR is passed on so that the DBaaS makefile installs the charts of the qubership-dbaas
# clone made here, at DBAAS_REPO_BRANCH. Left to itself it clones the default branch of
# qubership-dbaas into a directory of its own, and installs the aggregator chart from that, whatever
# DBAAS_REPO_BRANCH is: its scripts, its chart and its image would then come from three versions.
define run_dbaas_operation
	set -e; \
	echo "$(CYAN)--- Using config file: $(DBAAS_CONFIG_FILE)$(NC)"; \
	echo "$(CYAN)--- Using dbaas service name: $(DBAAS_SERVICE_NAME)$(NC)"; \
	echo "$(CYAN)--- Using dbaas namespace: $(DBAAS_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using pg namespace: $(PG_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using dbaas-aggregator tag: $(DBAAS_TAG)$(NC)"; \
	echo "$(CYAN)--- Using M2M auth mode: $(DBAAS_M2M_AUTH_MODE)$(NC)"; \
	echo "$(CYAN)--- Using dbaas operator enabled: $(DBAAS_OPERATOR_ENABLED)$(NC)"; \
	SKIP_CRDS=$(if $(filter true,$(INSTALL_CRDS)),false,true); \
	echo "$(CYAN)--- Using skip crds: $$SKIP_CRDS$(NC)"; \
	if ('$(MAKE)' -C .tmp/qubership-dbaas/bootstrap  $(1) CONFIG_FILE=$(DBAAS_CONFIG_FILE) PATRONI_SERVICES_VALUES_FILE=../../../dbaas/patroni-services-values.yaml DBAAS_SERVICE_NAME=$(DBAAS_SERVICE_NAME) DBAAS_NAMESPACE=$(DBAAS_NAMESPACE) PG_NAMESPACE=$(PG_NAMESPACE) SKIP_CRDS=$$SKIP_CRDS CREATE_NAMESPACE=$(CREATE_NAMESPACE) DBAAS_OPERATOR_ENABLED=$(DBAAS_OPERATOR_ENABLED) TAG=$(DBAAS_TAG) M2M_AUTH_MODE=$(DBAAS_M2M_AUTH_MODE) REPOS_DIR=$(abspath $(REPOS_DIR))); then \
		echo "$(GREEN)=== DBaaS $(1) completed successfully$(NC)"; \
	else \
		echo "$(RED)Error: DBaaS $(1) failed$(NC)"; \
		exit 1; \
	fi
endef

# Helper function to run MAAS operations
define run_maas_operation
	set -e; \
	echo "$(CYAN)--- Using config file: $(MAAS_CONFIG_FILE)$(NC)"; \
	echo "$(CYAN)--- Using MAAS branch or tag: $(MAAS_BRANCH)$(NC)"; \
	echo "$(CYAN)--- Using MAAS namespace: $(MAAS_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using monitoring: $(MONITORING_ENABLED)$(NC)"; \
	echo "$(CYAN)--- Using M2M auth mode: $(MAAS_M2M_AUTH_MODE)$(NC)"; \
	echo "$(CYAN)--- Using RabbitMQ namespace: $(RABBIT_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using Kafka namespace: $(KAFKA_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using Kafka instances: $(KAFKA_INSTANCES)$(NC)"; \
	echo "$(CYAN)--- Using RabbitMQ instances: $(RABBIT_INSTANCES)$(NC)"; \
	echo "$(CYAN)--- Using DBAAS namespace: $(DBAAS_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using DBAAS service name: $(DBAAS_SERVICE_NAME)$(NC)"; \
	if (cd maas && '$(MAKE)' $(1) CONFIG_FILE=$(MAAS_CONFIG_FILE) MAAS_BRANCH=$(MAAS_BRANCH) MAAS_NAMESPACE=$(MAAS_NAMESPACE) RABBIT_NAMESPACE=$(RABBIT_NAMESPACE) KAFKA_NAMESPACE=$(KAFKA_NAMESPACE) KAFKA_INSTANCES='$(KAFKA_INSTANCES)' RABBIT_INSTANCES='$(RABBIT_INSTANCES)' CREATE_NAMESPACE=$(CREATE_NAMESPACE) DBAAS_NAMESPACE=$(DBAAS_NAMESPACE) DBAAS_SERVICE_NAME=$(DBAAS_SERVICE_NAME) MONITORING_ENABLED=$(MONITORING_ENABLED) M2M_AUTH_MODE=$(MAAS_M2M_AUTH_MODE)); then \
		echo "$(GREEN)=== MAAS $(1) completed successfully$(NC)"; \
	else \
		echo "$(RED)Error: MAAS $(1) failed$(NC)"; \
		exit 1; \
	fi
endef

# Helper function to run core-mesh-config operations
# SERVICE_MESH_TYPE: when Istio is installed use Istio (even if profile says Core);
# otherwise propagate Core.
define run_core_mesh_config_operation
	set -e; \
	echo "$(CYAN)--- Using core-mesh-config config file: core-mesh-config/$(MESH_CONFIG_FILE)$(NC)"; \
	echo "$(CYAN)--- Using Istio namespace: $(ISTIO_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using core-mesh-config branch: $(CORE_MESH_CONFIG_REPO_BRANCH)$(NC)"; \
	echo "$(CYAN)--- Enable mesh for namespaces: $(CORE_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using SERVICE_MESH_TYPE: $(if $(filter true,$(INSTALL_ISTIO)),Istio,Core)$(NC)"; \
	if (cd core-mesh-config && '$(MAKE)' $(1) CONFIG_FILE=$(MESH_CONFIG_FILE) \
		ISTIO_NAMESPACE=$(ISTIO_NAMESPACE) \
		MESH_NAMESPACES=$(CORE_NAMESPACE) \
		CORE_MESH_CONFIG_REPO_BRANCH=$(CORE_MESH_CONFIG_REPO_BRANCH) \
		MONITORING_ENABLED=$(MONITORING_ENABLED) \
		SERVICE_MESH_TYPE=$(if $(filter true,$(INSTALL_ISTIO)),Istio,Core)); then \
		echo "$(GREEN)=== Core mesh config $(1) completed successfully$(NC)"; \
	else \
		echo "$(RED)Error: Core mesh config $(1) failed$(NC)"; \
		exit 1; \
	fi
endef

# Helper function to run Istio operations
define run_istio_operation
	set -e; \
	echo "$(CYAN)--- Using istio config file: istio/$(ISTIO_CONFIG_FILE)$(NC)"; \
	echo "$(CYAN)--- Using Istio namespace: $(ISTIO_NAMESPACE)$(NC)"; \
	echo "$(CYAN)--- Using Istio repo branch: $(ISTIO_REPO_BRANCH)$(NC)"; \
	SKIP_CRDS=$(if $(filter true,$(INSTALL_CRDS)),false,true); \
	echo "$(CYAN)--- Using skip crds: $$SKIP_CRDS$(NC)"; \
	if (cd istio && '$(MAKE)' $(1) CONFIG_FILE=$(ISTIO_CONFIG_FILE) \
		ISTIO_NAMESPACE=$(ISTIO_NAMESPACE) \
		SKIP_CRDS=$$SKIP_CRDS \
		CREATE_NAMESPACE=$(CREATE_NAMESPACE) \
		ISTIO_REPO_BRANCH=$(ISTIO_REPO_BRANCH) \
		MONITORING_ENABLED=$(MONITORING_ENABLED)); then \
		echo "$(GREEN)=== Istio $(1) completed successfully$(NC)"; \
	else \
		echo "$(RED)Error: Istio $(1) failed$(NC)"; \
		exit 1; \
	fi
endef
