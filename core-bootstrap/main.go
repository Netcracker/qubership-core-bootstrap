package main

import (
	"context"
	"flag"
	"os"

	"github.com/netcracker/core-bootstrap/v2/taskmanager/factory"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
)

var (
	logger = logging.GetLogger("main")
	ctx    = context.Background()
)

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

	taskManager := factory.CreateDefaultManager()

	if err := taskManager.Execute(ctx, isPostDeployPhase); err != nil {
		logger.PanicC(ctx, "Error during execution: %s", err)
	}
}
