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
    repository: "https://netcracker.github.io/qubership-core-bootstrap/"
```

2. Include the templates in your chart by creating a `corebootstrap.yaml` file in the `templates` directory with the following content:
```yaml
{{ include "coretpl.synchronizer.hooks" . }}
```

3. To declare databases for the DBaaS Operator, take `spec.operatorNamespace` of the CRs from the
   `coretpl.dbaasNamespace` template. It returns the namespace part of `API_DBAAS_ADDRESS`, the second label
   of the host, and fails the render when that label is missing, empty, or all digits, as in an IP address:
```yaml
spec:
  operatorNamespace: {{ include "coretpl.dbaasNamespace" $ | quote }}
```

4. Render the DBaaS Operator CRs, and the mount of the Secret the operator creates, only where the cluster serves the
   operator API, so the chart still installs on clusters without it. `coretpl.dbaasOperatorEnabled` returns `true`
   there and an empty string elsewhere:
```yaml
{{- if include "coretpl.dbaasOperatorEnabled" $ }}
apiVersion: dbaas.netcracker.com/v1
kind: InternalDatabase
...
{{- end }}
```

5. `coretpl.synchronizer.hooks` renders the Consul access of the service where `consulacl.enabled` returns `true`:
   in the Consul auth modes `hybrid` and `k8s`, on a cluster that serves the `ConsulACL` CRD. It renders:
   - a `ConsulACL` with the role `<NAMESPACE>_<SERVICE_NAME>_v2`: write to `config/<NAMESPACE>/<SERVICE_NAME>`, read of
     `logging/<NAMESPACE>/<SERVICE_NAME>`, and the shared `_v2` policies of cloud-core-configuration. Its binding rule
     gives the role to the ServiceAccount named `SERVICE_NAME`; a pod under `default` gets no role and cannot log in;
   - a `ConsulKV` for `config/`, `data/`, and `logging/<NAMESPACE>/<SERVICE_NAME>/`. Deleting it deletes these trees.

   A service that needs other rights sets `CORE_TPL_MS_CONSULACL_ENABLED: false` and renders its own resources. It
   takes `spec.operatorNamespace` from `consulacl.operatorNamespace`, the namespace part of `CONSUL_URL`.
   `coretpl.consulAuthMode` returns the mode and fails the render on a value other than `legacy`, `hybrid`, or `k8s`:
```yaml
{{- if include "consulacl.enabled" . }}
apiVersion: netcracker.com/v1
kind: ConsulACL
spec:
  operatorNamespace: {{ include "consulacl.operatorNamespace" . | quote }}
...
{{- end }}
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
| `API_DBAAS_ADDRESS` | DBaaS aggregator address, `http://<aggregator>.<namespace>:<port>`. Needed only by `coretpl.dbaasNamespace` |
| `CONSUL_URL` | Consul address, `http://<consul>.<namespace>:<port>`. Needed where `consulacl.enabled` returns `true` |
| `CONSUL_AUTH_MODE` | Consul auth mode: `legacy`, `hybrid`, or `k8s`. Needed by `coretpl.consulAuthMode` and `consulacl.enabled` |
| `M2M_AUTH_MODE` | Consul auth mode when `CONSUL_AUTH_MODE` is empty; `legacy` when both are empty |
| `ORIGIN_NAMESPACE`, `PEER_NAMESPACE` | Name the `<namespace>_read_bluegreen_v2` policies (read of `bluegreen/`) of the Consul role; `PEER_NAMESPACE` is optional |
| `CORE_TPL_MS_CONSULACL_ENABLED` | `false` turns off the Consul access of step 5; any other value or unset keeps it |

## Version Information

All library versions are available in the [Helm Repository](https://netcracker.github.io/qubership-core-bootstrap/index.yaml)

## How to release new version

To release a new version of the library, follow these steps:

1. Increase the version number in `Chart.yaml`
2. Push your changes to the `main` branch
3. Trigger the [Coretpl Helm Library Release](https://github.com/Netcracker/qubership-core-bootstrap/actions/workflows/coretpl---publish-release.yaml) workflow

The workflow will automatically build and publish the new version to the Helm repository.

