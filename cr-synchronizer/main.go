package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/netcracker/cr-synchronizer/getters"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	go func() {
		var isPostDeployPhase bool
		flag.BoolVar(&isPostDeployPhase, "post", false, "use cr-synchronizer as post-deploy waiter")
		flag.Parse()
		timeoutSeconds := 300 // 5min
		if val, ok := os.LookupEnv("RESOURCE_POLLING_TIMEOUT"); ok {
			if parsed, err := strconv.Atoi(val); err == nil {
				timeoutSeconds = parsed
			}
		}
		getters.StartGenerator(ctx, isPostDeployPhase, timeoutSeconds)
	}()

	<-ctx.Done()
	fmt.Println("received shutdown signal, exiting...")
}
