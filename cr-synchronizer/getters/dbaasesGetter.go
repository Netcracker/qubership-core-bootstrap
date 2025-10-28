package getters

import (
	"context"
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
	schemeRes, listRes := ng.declarationCreator(ng.resources, dbaasPlural)
	log.Info().Str("type", "creator").Str("kind", "dbaas").Msgf("finished declarationCreator")
	for _, declarativeName := range listRes {
		log.Info().Str("type", "waiter").Str("kind", "dbaas").Str("name", declarativeName).Msgf("starting declarationWaiter")
		ng.declarationWaiter(schemeRes, declarativeName)
		log.Info().Str("type", "waiter").Str("kind", "dbaas").Str("name", declarativeName).Msgf("finished declarationWaiter")
	}
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
