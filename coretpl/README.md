# CoreTPL

A Cloud-Core Uniform Library for Kubernetes deployments.

## Overview

CoreTPL is a Helm library chart designed to provide standardized templates and utilities for deploying applications in Kubernetes environments. It includes functionality for resource synchronization and deployment management.

## Usage

To integrate this library into your Helm chart, follow these steps:

1. Add CoreTPL as a dependency in your `Chart.yaml`:
```yaml
dependencies:
  - name: coretpl
    version: "<current version>"
    repository: "https://netcracker.github.io/qubership-core-bootstrap/index.yaml"
```

2. Include the templates in your chart by creating a `corebootstrap.yaml` file in the `templates` directory with the following content:
```yaml
{{ include "coretpl.synchronizer.hooks" . }}
```

## Configuration

The library requires the following configuration parameters:

| Parameter | Description                                           |
|-----------|-------------------------------------------------------|
| `DEPLOYMENT_SESSION_ID` | Unique identifier for the deployment session          |
| `APPLICATION_NAME` | Name of the application                               |
| `SERVICE_NAME` | Name of the service being deployed                    |
| `NAMESPACE` | Target deployment namespace                           |
| `CR_SYNCHRONIZER_IMAGE` | Image for the CR synchronizer                         |
| `RESOURCE_POLLING_TIMEOUT` | Timeout for resource polling in seconds (default: 300) |

## Version Information

All library versions are available in the [Helm Repository](https://netcracker.github.io/qubership-core-bootstrap/index.yaml)
