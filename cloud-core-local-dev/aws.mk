# Configuration file for cloud-core-local-dev on AWS
#
# This file only sets the values that differ from local.mk, and then includes local.mk for all the
# others, so a new parameter, a default or a comment in local.mk reaches the AWS profile without a
# second edit. The values here come first and use ?=, as those of local.mk do: the first definition
# wins, so a value set on the command line or in the environment still overrides them, and local.mk
# only fills in what is left.

# Installation: no namespace, CRDs, metrics server or consul to install, and MaaS is installed
CREATE_NAMESPACE ?= false
INSTALL_CRDS ?= false
INSTALL_METRICS_SERVER ?= false
INSTALL_CONSUL ?= false
INSTALL_MAAS ?= true

# Config files of the components installed by their own makefiles, which have an aws.mk of their own
DBAAS_CONFIG_FILE ?= aws.mk
MAAS_CONFIG_FILE ?= aws.mk
ISTIO_CONFIG_FILE ?= aws.mk
MESH_CONFIG_FILE ?= aws.mk

# Namespaces
CORE_NAMESPACE ?= core-1-core
PG_NAMESPACE ?= core-1-postgres
DBAAS_NAMESPACE ?= core-1-dbaas
MAAS_NAMESPACE ?= core-1-maas
RABBIT_NAMESPACE ?= core-1-maas
KAFKA_NAMESPACE ?= core-1-maas

# General values
DEPLOYMENT_SESSION_ID ?= cloud-core-aws-dev
CONSUL_SERVICE_NAME ?= consul-server

# MaaS
KAFKA_INSTANCES ?= kafka-1 kafka-2
RABBIT_INSTANCES ?= rabbit-1

# Core bootstrap
CORE_CONFIG_MAAS_ENABLED ?= true

# Everything else is the same as locally. The path is taken from this file's own location.
include $(dir $(lastword $(MAKEFILE_LIST)))local.mk
