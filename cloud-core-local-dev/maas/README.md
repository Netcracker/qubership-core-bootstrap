# MaaS Installation Script

This repository contains a comprehensive installation script for deploying Messaging-as-a-Service (MaaS) using Helm and Kubernetes.
The script manages the installation and uninstallation of Maas Service, RabbitMQ, Kafka components.

## Quick start
### 1. Prepare config file
Script distributed with 2 prepared configurations:
- local.mk - for local deployment (1 Kafka and 1 RabbitMQ instance)
- integration-tests.mk - for integration tests (2 Kafka and 2 RabbitMQ instances)

### 2. Execute installation command

#### minikube installation
`make install CONFIG_FILE=local.mk`
`make install CONFIG_FILE=integration-tests.mk`

local.mk applied by default

## Usage
Usage: `make <target> [CONFIG_FILE=local.mk]`

Targets:
- install            - Install all MaaS components
- uninstall          - Uninstall all MaaS components
- validate           - Validate configuration and prerequisites
- show-config        - Show current configuration
- cleanup-namespaces - Cleanup namespaces
- clean              - Clean up repositories
- help               - Show this help message

Examples:
- `make install CONFIG_FILE=local.mk`
- `make uninstall CONFIG_FILE=local.mk`
- `make validate CONFIG_FILE=local.mk`

## Profiles and Instance Count

The number of Kafka and RabbitMQ instances is controlled by variables in the profile file:

```
KAFKA_INSTANCES = kafka-1 kafka-2
RABBIT_INSTANCES = rabbitmq-1 rabbitmq-2
```

or for minimal profile:

```
KAFKA_INSTANCES = kafka-1
RABBIT_INSTANCES = rabbitmq-1
```

## Prerequisites

Before running the installation script, ensure you have the following tools installed and configured:

### Required Tools

1. **Helm**
2. **kubectl**
3. **git**

### Kubernetes Cluster Requirements

- A running Kubernetes cluster.
- DBaaS installed
- (optional) Existing namespaces. Can be created during installation - requires respective priveleges

### Namespaces

Specify in configuration 4 namespaces:
```DBAAS_NAMESPACE, RABBIT_NAMESPACE, KAFKA_NAMESPACE, DBAAS_NAMESPACE ```

Set ```CREATE_NAMESPACE=true```, if you want to create namespaces during installation (you need to have sufficient cluster privileges)
Set ```CREATE_NAMESPACE=false```, if you have pre-created namespaces

### Required permissions

Minimum permissions for installation (without admin privileges to create namespaces and CRDs)

- Ensure namespace admin privileges for DBAAS_NAMESPACE, RABBIT_NAMESPACE, KAFKA_NAMESPACE, DBAAS_NAMESPACE
- Ensure to have role

```
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

The script uses a .mk configuration file to define all installation and helm packages parameters

### Parameter Reference Table

| Parameter Name                           | Example Value                        | Description                                            |
|------------------------------------------|--------------------------------------|--------------------------------------------------------|
| **Configuration Files**                  |
| `MAAS_VALUES_FILE`                       | `./maas-values.yaml`                 | Path to MaaS Helm values file                          |
| `MAAS_PROFILE_FILE`                      | `./resource-profiles/dev.yaml`       | Path to resource profile file                          |
| **Namespace Configuration**              |
| `MAAS_NAMESPACE`                         | `maas`                               | Kubernetes namespace for MaaS                          |
| `RABBIT_NAMESPACE`                       | rabbit                               | Kubernetes namespace for RabbitMQ                      |
| `KAFKA_NAMESPACE`                        | kafka                                | Kubernetes namespace for Kafka                         |
| `DBAAS_NAMESPACE`                        | dbaas                                | Kubernetes namespace for DBaaS                         |
| **MaaS Configuration**                   |
| `TAG`                                    | `latest`                             | Docker image tag for MaaS                              |
| `DBAAS_SERVICE_NAME`                     | `dbaas-aggregator`                   | DBaaS service name                                     |
| `DBAAS_AGGREGATOR_ADDRESS`               | `http://dbaas-aggregator.dbaas:8080` | DBaaS aggregator URL in cluster                        |
| **Credentials**                          |
| `MAAS_ACCOUNT_MANAGER_USERNAME`          | `manager`                            | MaaS account manager username to set                   |
| `MAAS_ACCOUNT_MANAGER_PASSWORD`          | `manager`                            | MaaS account manager password to set                   |
| `MAAS_DEPLOYER_CLIENT_USERNAME`          | `client`                             | MaaS deployer clien username to set                    |
| `MAAS_DEPLOYER_CLIENT_PASSWORD`          | `client`                             | MaaS deployer clien password to set                    |
| `DBAAS_CLUSTER_DBA_CREDENTIALS_USERNAME` | `cluster-dba`                        | DBaaS cluster DBA username (to register MaaS database) |
| `DBAAS_CLUSTER_DBA_CREDENTIALS_PASSWORD` | `password`                           | DBaaS cluster DBA password (to register MaaS database) |
| **Installation Options**                 |
| `CREATE_NAMESPACE`                       | `true`                               | Automatically create namespaces if they don't exist    |
| `KAFKA_INSTANCES`                        | `kafka-1 kafka-2`                    | List of Kafka instance names to install                |
| `RABBIT_INSTANCES`                       | `rabbitmq-1 rabbitmq-2`              | List of RabbitMQ instance names to install             |

### Values Files

The script uses one main values file template:

**maas-values.yaml** - Configuration for MaaS component

The script uses `envsubst` to substitute environment variables in the values files templates

## Installation Process

The installation script performs the following stages:

### Stage 1: Repository Setup
- Clones required repositories:
    - `qubership-maas` from GitHub
      This is required as there is no public repo for helm packages downloading

### Stage 2: MaaS Service Installation
- Installs MaaS using Helm
- Uses the `MAAS_VALUES_FILE` configuration

### Stage 3: Wait for MaaS service started
- Waits for MaaS pod to be ready (timeout: 5 minutes)
- Ensures 1 pod is ready and MaaS server is started

### Stage 4: RabbitMQ Installation
- Installs RabbitMQ chart(s) using Helm for each name in `RABBIT_INSTANCES`

### Stage 5: Wait for RabbitMQ server started
- Waits for all RabbitMQ deployments to be ready (timeout: 5 minutes)
- Ensures all pods from `RABBIT_INSTANCES` are ready

### Stage 6: Kafka Installation
- Installs Kafka chart(s) using Helm for each name in `KAFKA_INSTANCES`

### Stage 7: Wait for Kafka server started
- Waits for all Kafka deployments to be ready (timeout: 5 minutes)
- Ensures all pods from `KAFKA_INSTANCES` are ready

### Stage 8: Register RabbitMQ instances
- Registers all RabbitMQ instances from `RABBIT_INSTANCES` in MaaS
- Uses `scripts/register-rabbit-instance-in-maas.sh` script

### Stage 9: Register Kafka instances
- Registers all Kafka instances from `KAFKA_INSTANCES` in MaaS
- Uses `scripts/register-kafka-instance-in-maas.sh` script

## Uninstallation Process

The uninstallation script performs the following stages:

### Stage 1: Uninstall MaaS service
- Removes MaaS Helm release

### Stage 2: Delete MaaS database
- Deletes MaaS database from DBaaS
- Uses `scripts/remove-maas-database-from-dbaas.sh` script

### Stage 3: Uninstall RabbitMQ
- Removes all RabbitMQ Helm releases from `RABBIT_INSTANCES`

### Stage 4: Uninstall Kafka
- Removes all Kafka Helm releases from `KAFKA_INSTANCES`

### Stage 5: Cleanup namespaces
- Deletes namespaces `MAAS_NAMESPACE`, `RABBIT_NAMESPACE`, `KAFKA_NAMESPACE` if `CREATE_NAMESPACE=true`