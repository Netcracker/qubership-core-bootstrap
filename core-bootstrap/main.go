package main

import (
	"context"
	"flag"
	"github.com/netcracker/core-bootstrap/v2/scripts/configserver"
	"github.com/netcracker/core-bootstrap/v2/scripts/consul"
	"github.com/netcracker/core-bootstrap/v2/scripts/controlplane"
	"github.com/netcracker/core-bootstrap/v2/scripts/dbaas"
	"github.com/netcracker/core-bootstrap/v2/scripts/maas"
	"github.com/netcracker/core-bootstrap/v2/scripts/staticcoregateway"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"os"
	"reflect"
)

var (
	logger = logging.GetLogger("main")
	ctx    = context.Background()
)

type TaskExecutor interface {
	Configure(func(string) string) error
	Execute(context.Context) error
}

func main() {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	utils.RegisterShutdownHook(func(exitCode int) {
		cancel()
		os.Exit(exitCode)
	})

	var isPostDeployPhase bool
	flag.BoolVar(&isPostDeployPhase, "post", false, "postdeploy")
	flag.Parse()

	if isPostDeployPhase {
		logger.InfoC(ctx, "Starting bootstrap postdeploy")
		postdeploy()
		logger.InfoC(ctx, "Finishing bootstrap postdeploy")
	} else {
		logger.InfoC(ctx, "Starting bootstrap predeploy")
		predeploy()
		logger.InfoC(ctx, "Finishing bootstrap predeploy")
	}

}

func predeploy() {
	consulConfigurer := consul.New()
	dbaasConfigurer := dbaas.New()

	tasks := []TaskExecutor{
		consulConfigurer,
		dbaasConfigurer,
		controlplane.New(dbaasConfigurer.CreateDatabase),
		configserver.New(consulConfigurer),
		maas.New(),
	}

	for _, task := range tasks {
		logger.InfoC(ctx, "Configure task: %s.%s", reflect.TypeOf(task).Elem().PkgPath(), reflect.TypeOf(task).Elem().Name())
		if err := task.Configure(os.Getenv); err != nil {
			logger.PanicC(ctx, "Error during configuring task %s: %s", reflect.TypeOf(task).Elem().PkgPath(), err)
		}
	}

	for _, task := range tasks {
		logger.InfoC(ctx, "Execute task: %s.%s", reflect.TypeOf(task).Elem().PkgPath(), reflect.TypeOf(task).Elem().Name())
		if err := task.Execute(ctx); err != nil {
			logger.PanicC(ctx, "Error during executing task %s: %s", reflect.TypeOf(task).Elem().PkgPath(), err)
		}
	}
}

func postdeploy() {
	tasks := []TaskExecutor{
		staticcoregateway.New(),
	}

	for _, task := range tasks {
		logger.InfoC(ctx, "Configure task: %s.%s", reflect.TypeOf(task).Elem().PkgPath(), reflect.TypeOf(task).Elem().Name())
		if err := task.Configure(os.Getenv); err != nil {
			logger.PanicC(ctx, "Error during configuring task %s: %s", reflect.TypeOf(task).Elem().PkgPath(), err)
		}
	}

	for _, task := range tasks {
		logger.InfoC(ctx, "Execute task: %s.%s", reflect.TypeOf(task).Elem().PkgPath(), reflect.TypeOf(task).Elem().Name())
		if err := task.Execute(ctx); err != nil {
			logger.PanicC(ctx, "Error during executing task %s: %s", reflect.TypeOf(task).Elem().PkgPath(), err)
		}
	}
}
