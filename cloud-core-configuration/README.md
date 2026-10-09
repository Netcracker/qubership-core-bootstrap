# cloud-core-configuration

This repository is intended to store the following Cloud Core components:
* Application chart

## Application chart

App chart for Cloud Core is intended to run project predeploy script before service charts processing. For this cause, the following entities were created:

* ServiceAccount with Role and RoleBinding
* Secret for storing environment variables which were lately propagated to bootstrap image
* Job with hook which runs project_predeploy.sh script stored inside image
* Persistent `dbaas-operator-secrets` Role and RoleBinding so dbaas-operator can manage
  Secrets in this namespace (`DatabaseSecretClaim` / `ExternalDatabase`). The RoleBinding
  subject namespace is taken from `API_DBAAS_ADDRESS`.

The chart depends on `coretpl` from this repository (`file://../../coretpl`) for the DBaaS Operator capability guard
and namespace. Run `helm dependency update` in `helm-charts` before installing it; `cloud-core-local-dev` does this
itself. Where the cluster serves the DBaaS Operator API, an `API_DBAAS_ADDRESS` without a namespace fails the render.
