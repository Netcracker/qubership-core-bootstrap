package main

import (
	"context"
	"flag"
	"github.com/netcracker/cr-synchronizer/getters"
	"os"
	"strconv"
)

func main() {
	ctx := context.Background()
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
}
