# cloud-core-configuration

This repository is intended to store the following Cloud Core components:
* Application chart

## Application chart

App chart for Cloud Core is intended to run project predeploy script before service charts processing. For this cause, the following entities were created:
* ServiceAccount with Role and RoleBinding
* Secret for storing environment variables which were lately propagated to bootstrap image
* Job with hook which runs project_predeploy.sh script stored inside image