# =============================================================================
# EARLY MESH
# =============================================================================
#
# Opt-in. Puts namespaces in the Istio ambient mesh, behind a waypoint, before the services in them
# start talking. Suites that must see or police every request, such as the M2M traffic map and the
# DBaaS Operator tests in qubership-core-test-apps, need this; a normal install does not.
#
# ---------------------------------------------------------------------------------------------
# CORE_MESH_EARLY=true: the core namespace is in the mesh for the whole of deploy-core
# ---------------------------------------------------------------------------------------------
#
# By default, the core-mesh-config chart, waypoint included, is installed before Cloud Core, but the
# core namespace is labeled for the mesh only at the very end of deploy-core, so nothing that happens
# during the install passes a waypoint:
#
#   default            ensure-core-namespace   core services and hooks start   enable-core-mesh
#                      | core-mesh-config chart|                               |  label namespace
#                      |  install waypoint     |                               |
#   core namespace     |- not in the mesh -----|---- not in the mesh ----------|- in the mesh ---->
#
#   CORE_MESH_EARLY    ensure-core-namespace   core services and hooks start   enable-core-mesh
#                      | enable-early-core-mesh|                               |  (changes nothing:
#                      |  label namespace      |                               |   the label is set)
#                      |  create waypoint      |                               |
#                      | core-mesh-config chart|                               |
#                      |  adopt waypoint       |                               |
#   core namespace     |- in the mesh, behind the waypoint ------------------------------------>
#
# enable-early-core-mesh creates the waypoint core-mesh-config would create, under the same name
# (waypoint) and with the label core-mesh-config would set (istio.io/use-waypoint=waypoint). The
# Gateway carries the Helm ownership metadata of the core-mesh-config release, so when that release
# installs right after, Helm adopts the Gateway instead of refusing a resource it does not own.
# Adoption adds the chart's settings to the Gateway, and istiod replaces the waypoint pod once,
# before any core service starts; a suite that reads the waypoint log from before deploy-core must
# follow the log of every waypoint pod.
#
# ---------------------------------------------------------------------------------------------
# MESH_DEPENDENCY_NAMESPACES=<ns>,...: dependencies behind a waypoint
# ---------------------------------------------------------------------------------------------
#
# DBaaS and MaaS are not part of the mesh. To see or police the calls the core makes to them, a
# namespace listed here is put in the ambient mesh, with a waypoint for all of its services, at the
# end of deploy-dependencies. Pods that already run join the mesh without a restart.
#
# Both waypoints are called waypoint, in every namespace, so a suite can attach to them by name.
# Their manifests are in utils/early-mesh/: core-waypoint.yaml and dependency-waypoint.yaml.

# Must match WAYPOINT_NAME of the core-mesh-config chart and the label in core-mesh-config/Makefile
MESH_WAYPOINT_NAME := waypoint
# Must match MESH_HELM_RELEASE_NAME in core-mesh-config/local.mk
CORE_MESH_HELM_RELEASE_NAME ?= core-istio-mesh
# The waypoint manifests, next to this file
EARLY_MESH_ASSETS := $(patsubst %/,%,$(dir $(lastword $(MAKEFILE_LIST))))/early-mesh

.PHONY: enable-early-core-mesh deploy-dependency-waypoints uninstall-dependency-waypoints

enable-early-core-mesh:
	@if [ "$(CORE_MESH_EARLY)" != "true" ]; then \
		echo "$(CYAN)--- Skipping early mesh of the core namespace (CORE_MESH_EARLY is not true)...$(NC)"; \
	elif [ "$(INSTALL_ISTIO)" != "true" ]; then \
		echo "$(RED)Error: CORE_MESH_EARLY=true needs INSTALL_ISTIO=true$(NC)"; \
		exit 1; \
	else \
		set -e; \
		echo "$(GREEN)=== Putting the core namespace $(CORE_NAMESPACE) in the mesh before Cloud Core is deployed...$(NC)"; \
		kubectl label namespace "$(CORE_NAMESPACE)" istio.io/dataplane-mode=ambient --overwrite; \
		$(call apply_waypoint,core-waypoint.yaml,$(CORE_NAMESPACE)); \
		kubectl label namespace "$(CORE_NAMESPACE)" istio.io/use-waypoint=$(MESH_WAYPOINT_NAME) --overwrite; \
		echo "$(GREEN)=== Core namespace $(CORE_NAMESPACE) is in the mesh, behind waypoint $(MESH_WAYPOINT_NAME)$(NC)"; \
	fi
	@echo ""

deploy-dependency-waypoints:
	@if [ -z "$(MESH_DEPENDENCY_NAMESPACES)" ]; then \
		echo "$(CYAN)--- Skipping waypoints for the dependencies (MESH_DEPENDENCY_NAMESPACES is empty)...$(NC)"; \
	elif [ "$(INSTALL_ISTIO)" != "true" ]; then \
		echo "$(RED)Error: MESH_DEPENDENCY_NAMESPACES needs INSTALL_ISTIO=true$(NC)"; \
		exit 1; \
	else \
		set -e; \
		for ns in $$(echo "$(MESH_DEPENDENCY_NAMESPACES)" | tr ',' ' '); do \
			echo "$(GREEN)=== Putting namespace $$ns in the mesh, behind waypoint $(MESH_WAYPOINT_NAME)...$(NC)"; \
			kubectl label namespace "$$ns" istio.io/dataplane-mode=ambient --overwrite; \
			$(call apply_waypoint,dependency-waypoint.yaml,$$ns); \
			kubectl label namespace "$$ns" istio.io/use-waypoint=$(MESH_WAYPOINT_NAME) --overwrite; \
		done; \
	fi
	@echo ""

uninstall-dependency-waypoints:
	@for ns in $$(echo "$(MESH_DEPENDENCY_NAMESPACES)" | tr ',' ' '); do \
		echo "$(YELLOW)--- Taking namespace $$ns out of the mesh...$(NC)"; \
		kubectl label namespace "$$ns" istio.io/use-waypoint- istio.io/dataplane-mode- --overwrite 2>/dev/null || true; \
		kubectl -n "$$ns" delete gateway.gateway.networking.k8s.io/$(MESH_WAYPOINT_NAME) configmap/$(MESH_WAYPOINT_NAME)-pod-options --ignore-not-found=true || true; \
	done

# Applies a waypoint asset from utils/early-mesh/ to a namespace, then waits until istiod has
# programmed the waypoint and its pod runs.
# Usage: $(call apply_waypoint,<asset file>,<namespace>)
# Fully qualified kind: gateways.core.netcracker.com has the same short name.
define apply_waypoint
	WAYPOINT_NAME=$(MESH_WAYPOINT_NAME) RELEASE_NAME=$(CORE_MESH_HELM_RELEASE_NAME) RELEASE_NAMESPACE=$(2) \
		envsubst '$${WAYPOINT_NAME} $${RELEASE_NAME} $${RELEASE_NAMESPACE}' < $(EARLY_MESH_ASSETS)/$(1) \
		| kubectl -n "$(2)" apply -f -; \
	kubectl -n "$(2)" wait --for=condition=Programmed gateway.gateway.networking.k8s.io/$(MESH_WAYPOINT_NAME) --timeout=180s; \
	kubectl -n "$(2)" rollout status deployment/$(MESH_WAYPOINT_NAME) --timeout=180s
endef
