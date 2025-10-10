package getters

import (
	"context"
	ncapi "github.com/netcracker/cr-synchronizer/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

const (
	maasesRunner = "maasDeclarativeClient"
	maasPlural   = "maases"
)

type MaaSesRunner struct {
	resources []unstructured.Unstructured
	DeploymentGenerator
}

func (ng *MaaSesRunner) Generate() {
	log.Info().Str("type", "creator").Str("kind", "maas").Msgf("starting declarationCreator")
	schemeRes, listRes := ng.declarationCreator(ng.resources, maasPlural)
	log.Info().Str("type", "creator").Str("kind", "maas").Msgf("finished declarationCreator")
	if len(listRes) > 0 {
		for _, declarativeName := range listRes {
			log.Info().Str("type", "waiter").Str("kind", "maas").Str("name", declarativeName).Msgf("starting declarationWaiter")
			ng.declarationWaiter(schemeRes, declarativeName)
			log.Info().Str("type", "waiter").Str("kind", "maas").Str("name", declarativeName).Msgf("finished declarationWaiter")
		}
	}
}

func NewMaaSesRunnerGenerator(ctx context.Context, resources []unstructured.Unstructured, client dynamic.Interface, recorder EventRecorder, clientset ncapi.Interface, scheme *runtime.Scheme, runtimeReceiver runtime.Object, timeoutSeconds int) *MaaSesRunner {
	return &MaaSesRunner{
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

func (ng *MaaSesRunner) Name() string {
	return maasesRunner
}
