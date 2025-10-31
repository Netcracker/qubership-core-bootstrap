# core-bootstrap-image

Core bootstrap image is dedicated to run predeploy scripts for cloud core app.
It runs after core validation image and before preinstall jobs for cloud-core microservices.

List of predeploy scripts:

1. maas config script - sends configuration declared by MAAS_CONFIG env to maas. common usage is put maas designators for rabbit and kafka
2. maas client creation script - used by maas agent to communicate with maas
3. dbaas autobalance scripts - 2 scripts for maas designators per namespace or per microservice
4. control plane prepare db - creates db for control plane
5. config server script - creates consult token and stores it in dedicated secret 
