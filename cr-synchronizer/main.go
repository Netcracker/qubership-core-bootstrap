package main

import (
	"flag"
	"github.com/netcracker/cr-synchronizer/getters"
)

func main() {
	var isPostDeployPhase bool
	flag.BoolVar(&isPostDeployPhase, "post", false, "use cr-synchronizer as post-deploy waiter")
	flag.Parse()

	getters.StartGenerator(isPostDeployPhase)
}
