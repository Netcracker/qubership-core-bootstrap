package config

import (
	"github.com/netcracker/core-bootstrap/v2/scripts/configserver"
	"github.com/netcracker/core-bootstrap/v2/scripts/consul"
	"github.com/netcracker/core-bootstrap/v2/scripts/controlplane"
	"github.com/netcracker/core-bootstrap/v2/scripts/dbaas"
	"github.com/netcracker/core-bootstrap/v2/scripts/maas"
	"github.com/netcracker/core-bootstrap/v2/scripts/staticcoregateway"
	"github.com/netcracker/core-bootstrap/v2/taskmanager"
)

func DefaultTasks() ([]taskmanager.TaskExecutor, []taskmanager.TaskExecutor) {
	consulConfigurer := consul.New()
	dbaasConfigurer := dbaas.New()

	preDeployTasks := []taskmanager.TaskExecutor{
		consulConfigurer,
		dbaasConfigurer,
		controlplane.New(dbaasConfigurer.CreateDatabase),
		configserver.New(consulConfigurer),
		maas.New(),
	}

	postDeployTasks := []taskmanager.TaskExecutor{
		staticcoregateway.New(),
	}

	return preDeployTasks, postDeployTasks
}
