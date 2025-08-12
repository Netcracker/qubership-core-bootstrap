package getters

import (
	"context"
	"sync"

	ncapi "github.com/netcracker/cr-synchronizer/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

const (
	dbaasesRunner = "dbaasDeclarativeClient"
	dbaasPlural   = "dbaases"
)

type DBaaSesRunner struct {
	resources []unstructured.Unstructured
	DeploymentGenerator
}

func (ng *DBaaSesRunner) Generate() {
	log.Info().Str("type", "creator").Str("kind", "dbaas").Msgf("starting declarationCreator")
	schemeRes := ng.declarationCreator(ng.resources, dbaasPlural)
	log.Info().Str("type", "creator").Str("kind", "dbaas").Msgf("finished declarationCreator")
	var wg sync.WaitGroup
	for resource, names := range schemeRes {
		for _, declarativeName := range names {
			log.Info().Str("type", "waiter").Str("kind", "dbaas").Str("name", declarativeName).Msgf("starting declarationWaiter")
			wg.Add(1)
			go func() {
				ng.declarationWaiter(resource, declarativeName)
				wg.Done()
			}()
			log.Info().Str("type", "waiter").Str("kind", "dbaas").Str("name", declarativeName).Msgf("finished declarationWaiter")
		}
	}
	wg.Wait()
}

func NewDBaaSesRunnerGenerator(ctx context.Context, resources []unstructured.Unstructured, client dynamic.Interface, recorder EventRecorder, clientset ncapi.Interface, scheme *runtime.Scheme, runtimeReceiver runtime.Object, timeoutSeconds int) *DBaaSesRunner {
	return &DBaaSesRunner{
		resources: resources,
		DeploymentGenerator: DeploymentGenerator{
			ctx:             ctx,
			client:          client,
			clientset:       clientset,
			recorder:        recorder,
			scheme:          scheme,
			runtimeReceiver: runtimeReceiver,
			timeoutSeconds:  timeoutSeconds,
		},
	}
}

func (ng *DBaaSesRunner) Name() string {
	return dbaasesRunner
}
