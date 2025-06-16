package getters

import (
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
	GoCrChecker(func() { ng.initialize() })
}

func NewMaaSesRunnerGenerator(resources []unstructured.Unstructured, client *dynamic.DynamicClient, recorder EventRecorder, clientset *ncapi.Clientset, scheme *runtime.Scheme, runtimeReceiver runtime.Object) *MaaSesRunner {
	return &MaaSesRunner{
		resources: resources,
		DeploymentGenerator: DeploymentGenerator{
			client:          client,
			clientset:       clientset,
			recorder:        recorder,
			scheme:          scheme,
			runtimeReceiver: runtimeReceiver,
		},
	}
}

func (ng *MaaSesRunner) Name() string {
	return maasesRunner
}

func (ng *MaaSesRunner) initialize() {
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
