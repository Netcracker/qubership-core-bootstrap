# Cloud Core Installation Script

This repository contains a comprehensive installation script for deploying Cloud Core components using Helm and Kubernetes. The script manages the installation and uninstallation of Cloud Core dependencies and services.

## Dependencies and Services

### Infrastructure Dependencies
- **Metrics Server** - [Optional] Kubernetes metrics collection and aggregation
- **Qubership Monitoring Operator** - [Optional] Automates the deployment and management of complete monitoring infrastructure on Kubernetes 
- **Consul**
- **DBaaS** - Database-as-a-Service with PostgreSQL and Patroni

### Core Cloud
- **Cloud Core Configuration** - Bootstrap configuration and initialization
- **Facade Operator**
- **Ingress Gateway**
- **Site Management**
- **PaaS Mediation**
- **Control Plane**
- **DBaaS Agent**
- **Core Operator**
- **Config Server**
- **MaaS** [Optional]

## Quick Start

### 1. Prepare Configuration File
The script is distributed with prepared configurations:
- `local.mk` - for local deployment (default)
- `aws.mk` - for AWS deployment

### 2. Execute Installation Command

#### Local Installation (minikube/rancher desktop/...)
```bash
make install CONFIG_FILE=local.mk
```
or simply:
```bash
make install
```
(local.mk is applied by default)

#### AWS Installation
```bash
make install CONFIG_FILE=aws.mk
```
## Usage

**Usage:** `make <target> [CONFIG_FILE=local.mk]`

### Main Targets

- **`install`** - Install all Cloud Core components and run mesh smoke test
- **`uninstall`** - Complete uninstall, CRDs and namespaces cleanup
- **`validate`** - Validate configuration and prerequisites
- **`mesh-smoke-test`** - Run mesh connectivity smoke test

### Examples

```bash
# Full installation
make install CONFIG_FILE=local.mk

# Uninstall everything
make uninstall CONFIG_FILE=local.mk

# Run mesh smoke test only
make mesh-smoke-test CONFIG_FILE=local.mk

# Complete cleanup
make cleanup-all CONFIG_FILE=local.mk

# Validate prerequisites
make validate CONFIG_FILE=local.mk
```

## Prerequisites

Before running the installation script, ensure you have the following tools installed and configured:

### Required Tools

1. **Helm** (v3.x)
2. **kubectl** (configured with cluster access)
3. **git**
4. **envsubst** (usually included with gettext)

### Kubernetes Cluster Requirements

- A running Kubernetes cluster (minikube, rancher-desktop, or cloud-based)
- Sufficient cluster privileges for namespace creation and CRD installation
- At least 3GB RAM and 2 CPU cores available for the cluster

### Namespaces

The script manages several namespaces:
- `CORE_NAMESPACE` - Main namespace for Cloud Core components
- `CONSUL_NAMESPACE` - Namespace for Consul service mesh
- `MONITORING_NAMESPACE` - Namespace for monitoring stack
- `DBAAS_NAMESPACE` - Namespace for DBaaS components
- `PG_NAMESPACE` - Namespace for PostgreSQL components
- `MAAS_NAMESPACE` - Namespace for MAAS components
- `RABBIT_NAMESPACE` - Namespace for RabbitMQ components
- `KAFKA_NAMESPACE` - Namespace for Kafka components

Set `CREATE_NAMESPACE=true` to automatically create namespaces during installation, or `CREATE_NAMESPACE=false` if you have pre-created namespaces.

### Required CRDs

The following Custom Resource Definitions are required:
- `facadeservices.qubership.org`
- `gateways.core.qubership.org`
- `securities.core.qubership.org`
- `meshes.core.qubership.org`
- `dbaases.core.qubership.org`
- `maases.core.qubership.org`
- `composites.core.qubership.org`

Set `INSTALL_CRDS=true` to install CRDs automatically, or `INSTALL_CRDS=false` to skip CRD installation (the script will validate their presence).

### Required Permissions

Minimum permissions for installation:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: qubership
rules:
- apiGroups:
  - "qubership.org"
  resources:
  - "*"
  verbs:
  - get
  - create
  - get
  - list
  - patch
  - update
  - watch 
  - delete
```

## Configuration

### Configuration File

The script uses a `.mk` configuration file to define all installation parameters and Helm package configurations.

### Installation Parameter Reference Table

| Parameter Name | Example Value | Description |
|---|---|---|
| **Installation Options (not propagetd to helm values)** |
| `CREATE_NAMESPACE` | `true`/`false` | Automatically create namespaces if they don't exist |
| `INSTALL_CRDS` | `true`/`false` | Install Custom Resource Definitions |
| `INSTALL_METRICS_SERVER` | `true`/`false` | Install Kubernetes metrics server |
| `INSTALL_MONITORING` | `true`/`false` | Install monitoring operator |
| `INSTALL_CONSUL` | `true`/`false` | Install Consul |
| `INSTALL_DBAAS` | `true`/`false` | Install DBaaS components |
| `DBAAS_CONFIG_FILE` | `local.mk` | DBaaS configuration file path (relative path will be resolved upon ./dbaas folder, where sub-Makefile is placed)|
| `INSTALL_MAAS` | `true`/`false` | Install MAAS components |
| `MAAS_CONFIG_FILE` | `local.mk` | MAAS configuration file path (relative path will be resolved upon ./maas folder, where sub-Makefile is placed)|

### Values Files

The script uses several values files:

1. **`core-values.yaml`** - Main configuration values for Cloud Core components (generated by script)
2. **`core-values.envsubst`** - Template for core values with environment variable substitution
3. **`core-bootstrap-helm-repo.yaml`** - Helm repository configuration

The script uses `envsubst` to substitute environment variables in the values file templates.

### MaaS configuration

Specify list of instance names in parameters `KAFKA_INSTANCES`, `RABBIT_INSTANCES` to adjust message brokers clusters

e.g. 
`KAFKA_INSTANCES ?= kafka-1 kafka-2` means deploy 2 instances of Kafka
`RABBIT_INSTANCES ?=` means do not deploy RabbitMQ at all

## Installation Process

The installation script performs the following stages:

### Stage 1: Validation
- Validates CLI tools (helm, kubectl, git)
- Validates cluster connectivity
- Validates CRD existence (if `INSTALL_CRDS=false`)
- Validates namespace existence (if `CREATE_NAMESPACE=false`)

### Stage 2: Repository Setup
- Clones required repositories:
  - `qubership-monitoring-operator`
  - `qubership-core-paas-mediation`
  - `qubership-core-control-plane`
  - `qubership-core-ingress-gateway`
  - `qubership-core-site-management`
  - `qubership-core-facade-operator`
  - `qubership-core-dbaas-agent`
  - `qubership-core-core-operator`
  - `qubership-core-config-server`
  - `consul-k8s` (if `INSTALL_CONSUL=true`)

### Stage 3: Values File Generation
- Generates `core-values.yaml` from `core-values.envsubst` template
- Substitutes environment variables

### Stage 4: Dependencies Installation
- **Metrics Server** (if `INSTALL_METRICS_SERVER=true`)
- **Monitoring** (if `INSTALL_MONITORING=true`)
- **Consul** (if `INSTALL_CONSUL=true`)
  - Clones Consul repository
  - Updates security context for local deployment
  - Installs Consul with custom configuration
  - Waits for Consul pods to be ready
  - Validates Consul connectivity
- **DBaaS** (if `INSTALL_DBAAS=true`)
  - Calls DBaaS installation sub-Makefile
  - Runs smoke test if installation is skipped

### Stage 5: Core Bootstrap
- Ensures core namespace exists
- Deploys cloud-core-configuration Helm chart

### Stage 6: Core Components Installation
- **Facade Operator**
  - Applies CRDs (if `INSTALL_CRDS=true`)
  - Updates Helm dependencies
  - Installs Facade Operator
- **Ingress Gateway**
  - Updates Helm dependencies
  - Installs Ingress Gateway
- **Site Management**
  - Updates Helm dependencies
  - Installs Site Management
- **PaaS Mediation**
  - Updates Helm dependencies
  - Installs PaaS Mediation
- **Control Plane**
  - Updates Helm dependencies
  - Installs Control Plane
- **DBaaS Agent**
  - Updates Helm dependencies
  - Installs DBaaS Agent
- **Core Operator**
  - Applies CRDs (if `INSTALL_CRDS=true`)
  - Updates Helm dependencies
  - Installs Core Operator
- **Config Server**
  - Updates Helm dependencies
  - Installs Config Server
- **MAAS** (if `INSTALL_MAAS=true`)
  - Calls MAAS installation sub-Makefile
  - Installs MAAS components with RabbitMQ and Kafka

### Stage 7: Mesh Smoke Test
- Deploys mesh test resources (`mesh-test.yaml`)
- Waits for test pod readiness
- Runs connectivity test through public gateway
- Cleans up test resources

## Uninstallation Process

The uninstallation script performs the following stages in reverse order:

### Stage 1: Uninstall Core Components
- Uninstalls Config Server
- Uninstalls Core Operator
- Uninstalls DBaaS Agent
- Uninstalls Control Plane
- Uninstalls PaaS Mediation
- Uninstalls Site Management
- Uninstalls Ingress Gateway
- Uninstalls Facade Operator
- Uninstalls cloud-core-configuration

### Stage 2: Uninstall Dependencies
- Uninstalls MAAS components (if `INSTALL_MAAS=true`)
- Uninstalls DBaaS components
- Uninstalls Consul (if `INSTALL_CONSUL=true`)
- Uninstalls monitoring (if `INSTALL_MONITORING=true`)
- Uninstalls metrics server (if `INSTALL_METRICS_SERVER=true`)

### Stage 3: Cleanup
- Cleans up mesh test resources
- Removes CRDs (if `INSTALL_CRDS=true`)
- Removes namespaces (if `CREATE_NAMESPACE=true`)

## Mesh Smoke Test

The mesh smoke test validates the connectivity and routing capabilities of the deployed Cloud Core components:

### Test Process
1. **Deploy Test Resources** - Applies `mesh-test.yaml` with test service and routes
2. **Wait for Readiness** - Waits up to 3 minutes for test pod to be ready
3. **Internal Connectivity Test** - Tests direct service-to-service communication
4. **External Gateway Test** - Tests access through the public gateway
5. **Cleanup** - Removes all test resources

### Debug Mode

Enable debug mode to see detailed Helm operations:
```bash
make install CONFIG_FILE=local.mk DEBUG=true
```

### Logs and Status

Check component status:
```bash
# Check all pods
kubectl get pods -n core

# Check specific component
kubectl get pods -n core -l app=facade-operator

# View logs
kubectl logs -n core -l app=facade-operator
```

## Repository Structure

```
cloud-core-local-dev/
├── Makefile                    # Main installation script
├── local.mk                    # Local deployment configuration
├── aws.mk                      # AWS deployment configuration
├── core-values.envsubst        # Core values template
├── core-bootstrap-helm-repo.yaml # Helm repository config
├── test/                       # Test resources
├── test/mesh/                  # Mesh smoke test resources
├── dbaas/                      # DBaaS installation subdirectory
├── maas/                       # MAAS installation subdirectory
├── repos/                      # [Created by script] Cloned repositories
└── minikube/                   # Script for minikube cluster preparation
```