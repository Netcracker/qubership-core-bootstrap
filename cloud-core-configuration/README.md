# cloud-core-configuration

This repository is intended to store the following Cloud Core components:
* Application chart

## Application chart

App chart for Cloud Core is intended to run project predeploy script before service charts processing. For this cause, the following entities were created:

* ServiceAccount with Role and RoleBinding
* Secret for storing environment variables which were lately propagated to bootstrap image
* Job with hook which runs project_predeploy.sh script stored inside image
* `dbaas-operator-secrets` Role and RoleBinding, rendered only where the cluster serves
  `dbaas.netcracker.com/v1`, so dbaas-operator can manage Secrets in this namespace
  (`DatabaseSecretClaim` / `ExternalDatabase`). Unlike the hook resources above, they stay
  installed. The RoleBinding subject namespace is taken from `API_DBAAS_ADDRESS`.

The chart depends on `coretpl` from this repository (`file://../../coretpl`) for the DBaaS Operator capability guard
and namespace, and for the Consul helpers. Run `helm dependency update` in `helm-charts` before installing it;
`cloud-core-local-dev` does this itself. Where the cluster serves the DBaaS Operator API, an `API_DBAAS_ADDRESS`
without a namespace fails the render.

Where `consulacl.enabled` of coretpl returns `true`, the chart renders the `ConsulACL` `cloud-core`: the shared `_v2`
policies the service roles use, and the binding rule that gives a ServiceAccount other than `default` the role
`<NAMESPACE>_<ServiceAccount>_v2`. It also renders a `ConsulKV` for `config/<NAMESPACE>/application/` and
`data/<NAMESPACE>/`; deleting the release deletes these trees. These resources need `CONSUL_URL` with the Consul
namespace in its host, `ORIGIN_NAMESPACE`, and, on a satellite, `BASELINE_ORIGIN`; without a namespace in `CONSUL_URL`
the render fails.
