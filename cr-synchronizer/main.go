package main

import (
	"flag"
	"github.com/netcracker/cr-synchronizer/getters"
	"os"
	"strconv"
)

func main() {
	var isPostDeployPhase bool
	flag.BoolVar(&isPostDeployPhase, "post", false, "use cr-synchronizer as post-deploy waiter")
	flag.Parse()
	timeoutSeconds := 30
	if val, ok := os.LookupEnv("RESOURCE_POLLING_TIMEOUT"); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			timeoutSeconds = parsed
		}
	}
	getters.StartGenerator(isPostDeployPhase, timeoutSeconds)
}
