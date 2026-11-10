# core-bootstrap-image

Core bootstrap image is dedicated to run predeploy scripts for cloud core app.
It runs after core validation image and before preinstall jobs for cloud-core microservices.

List of predeploy scripts:

1. maas config script - sends configuration declared by MAAS_CONFIG env to maas. common usage is put maas designators for rabbit and kafka
2. maas client creation script - used by maas agent to communicate with maas
3. dbaas autobalance scripts - 2 scripts for maas designators per namespace or per microservice
4. control plane prepare db - creates db for control plane
5. config server script - creates a Consul token for config-server and stores it in the `config-server-consul-token`
   secret. With `CONSUL_AUTH_MODE` set to `k8s` (or `M2M_AUTH_MODE`, when `CONSUL_AUTH_MODE` is empty), it deletes the
   secret instead, since core-legacy-api logs in to Consul itself. With `hybrid`, it deletes the secret when the Consul
   auth method `applications-k8s-m2m` accepts the audience `netcracker` or Consul has ACLs disabled, creates the
   token when the method is missing or lacks the audience, and fails on any other answer of Consul. Requires
   `CONSUL_ENABLED=true`.
