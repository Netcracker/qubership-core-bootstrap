package taskmanager

import (
	"context"
	"os"
	"reflect"

	"github.com/netcracker/qubership-core-lib-go/v3/logging"
)

var logger = logging.GetLogger("taskmanager")

type TaskExecutor interface {
	Configure(func(string) string) error
	Execute(context.Context) error
}

type TaskManager struct {
	preDeployTasks  []TaskExecutor
	postDeployTasks []TaskExecutor
}

func New(preDeployTasks, postDeployTasks []TaskExecutor) *TaskManager {
	return &TaskManager{
		preDeployTasks:  preDeployTasks,
		postDeployTasks: postDeployTasks,
	}
}

func (tm *TaskManager) executeTasks(ctx context.Context, tasks []TaskExecutor) error {
	for _, task := range tasks {
		logger.InfoC(ctx, "Configure task: %s.%s", reflect.TypeOf(task).Elem().PkgPath(), reflect.TypeOf(task).Elem().Name())
		if err := task.Configure(os.Getenv); err != nil {
			return err
		}
	}

	for _, task := range tasks {
		logger.InfoC(ctx, "Execute task: %s.%s", reflect.TypeOf(task).Elem().PkgPath(), reflect.TypeOf(task).Elem().Name())
		if err := task.Execute(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (tm *TaskManager) Execute(ctx context.Context, isPostDeployPhase bool) error {
	if isPostDeployPhase {
		logger.InfoC(ctx, "Starting postdeploy phase")
		return tm.executeTasks(ctx, tm.postDeployTasks)
	} else {
		logger.InfoC(ctx, "Starting predeploy phase")
		return tm.executeTasks(ctx, tm.preDeployTasks)
	}
}
