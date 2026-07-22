# Dependency versions for cloud-core-local-dev
# Override via: make DEPENDENCIES_FILE=dependencies.mk ...

# Image/helm tag for MaaS. "latest" checks out main; otherwise checks out the matching GitHub tag.
MAAS_TAG ?= v5.5.10
