package factory

import (
	"github.com/netcracker/core-bootstrap/v2/taskmanager"
	"github.com/netcracker/core-bootstrap/v2/taskmanager/config"
)

func CreateDefaultManager() *taskmanager.TaskManager {
	preDeployTasks, postDeployTasks := config.DefaultTasks()
	return taskmanager.New(preDeployTasks, postDeployTasks)
}

func CreateCustomManager(customPreDeployTasks, customPostDeployTasks []taskmanager.TaskExecutor) *taskmanager.TaskManager {
	defaultPreDeployTasks, defaultPostDeployTasks := config.DefaultTasks()

	allPreDeployTasks := append(defaultPreDeployTasks, customPreDeployTasks...)
	allPostDeployTasks := append(defaultPostDeployTasks, customPostDeployTasks...)

	return taskmanager.New(allPreDeployTasks, allPostDeployTasks)
}
