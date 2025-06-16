package getters

import (
	"fmt"
	v1alpha1 "github.com/netcracker/cr-synchronizer/api/types/v1"
	ncapi "github.com/netcracker/cr-synchronizer/clientset"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/pkgerrors"
	v1Core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	wr = diode.NewWriter(os.Stdout, 1000, 10*time.Millisecond, func(missed int) {
		fmt.Printf("Logger Dropped %d messages", missed)
	})

	output := zerolog.ConsoleWriter{
		NoColor:    true,
		Out:        wr,
		TimeFormat: time.RFC3339, // ISO 8601 format with milliseconds
		FormatTimestamp: func(i interface{}) string {
			return fmt.Sprintf("[%s]", i)
		},
		FormatLevel: func(i interface{}) string {
			return fmt.Sprintf("[%s]", i)
		},
		FormatFieldName: func(i interface{}) string {
			return fmt.Sprintf("[%v=", i)
		},
		FormatFieldValue: func(i interface{}) string {
			return fmt.Sprintf("%v]", i)
		},
		FormatMessage: func(i interface{}) string {
			return fmt.Sprintf("%v", i)
		},
		FieldsOrder: []string{"type", "name"},
	}

	zerolog.TimeFieldFormat = time.RFC1123
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log = zerolog.New(output).With().Timestamp().Caller().Logger()
	namespace = GetCurrentNS()
	serviceName = os.Getenv("SERVICE_NAME")
}

func StartGenerator(pst bool) {
	postdeploy = pst
	prepare()
	wr.Close()
}

type GeneratorManager struct {
	generators map[string]Generator
}

var generatorManager *GeneratorManager

func (gm *GeneratorManager) register(generator Generator) {
	if name := generator.Name(); name != "" {
		gm.generators[name] = generator
	}
}

func (gm *GeneratorManager) run() {
	//if any generator adds, runs fast, then done - allCrCheckersWaitGroup will be zero before second generator even started
	allCrCheckersWaitGroup.Add(1)
	for name, generator := range gm.generators {
		log.Info().Str("type", "generator").Str("name", name).Msgf("Register new waiter")
		generator.Generate()
	}
	allCrCheckersWaitGroup.Done()
}

func prepare() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("InCluster config can't be initialized")
	}
	v1alpha1.AddToScheme(scheme.Scheme)
	clientSet, err := ncapi.NewForConfig(config)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("ClientSet can't be initialized")
	}
	runtimeReceiver := setEventReceiver(clientSet)
	eventBroadcaster := NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: clientSet.CoreV1().Events(namespace)})
	recorder := eventBroadcaster.NewLabeledRecorder(
		scheme.Scheme,
		v1Core.EventSource{Component: coreDeclarativeEventsGenerator, Host: component_instance})
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Dynamic Client can't be initialized")
	}
	NewDeclarativeGenerator(client, recorder, clientSet, scheme.Scheme, runtimeReceiver).Run()

	log.Info().Str("type", "init").Msgf("generator finished")
}

type DeploymentGenerator struct {
	client          *dynamic.DynamicClient
	recorder        EventRecorder
	clientset       *ncapi.Clientset
	scheme          *runtime.Scheme
	runtimeReceiver runtime.Object
}

func NewDeclarativeGenerator(client *dynamic.DynamicClient, recorder EventRecorder, clientset *ncapi.Clientset, scheme *runtime.Scheme, runtimeReceiver runtime.Object) *DeploymentGenerator {
	return &DeploymentGenerator{
		client:          client,
		recorder:        recorder,
		clientset:       clientset,
		scheme:          scheme,
		runtimeReceiver: runtimeReceiver,
	}
}

func (ng *DeploymentGenerator) Run() {
	ng.initialize()
	WaitForAllCrCheckers()
}

func WaitForAllCrCheckers() {
	allCrCheckersWaitGroup.Wait()
}

func (ng *DeploymentGenerator) createGenericGeneratorManager() *GeneratorManager {
	generatorManager = &GeneratorManager{
		generators: make(map[string]Generator),
	}
	generatorManager.register(NewGenericRunnerGenerator(ng.client, ng.recorder, ng.clientset, ng.scheme, ng.runtimeReceiver))
	return generatorManager
}

func (ng *DeploymentGenerator) createKnownGeneratorManager(dcl map[string][]unstructured.Unstructured) *GeneratorManager {
	generatorManager = &GeneratorManager{
		generators: make(map[string]Generator),
	}
	generatorManager.register(NewMaaSesRunnerGenerator(dcl[MaaSKind], ng.client, ng.recorder, ng.clientset, ng.scheme, ng.runtimeReceiver))
	generatorManager.register(NewDBaaSesRunnerGenerator(dcl[DBaaSKind], ng.client, ng.recorder, ng.clientset, ng.scheme, ng.runtimeReceiver))
	return generatorManager
}

func (ng *DeploymentGenerator) initialize() {
	var generatorManager *GeneratorManager
	rptm, _ = strconv.Atoi(os.Getenv("RESOURCE_POLLING_TIMEOUT"))
	if !postdeploy {
		log.Info().Str("mode", "synchronizer").Msgf("Synchronizer hook started")
		installedDeclaratives := prepareDataFromFiles()
		generatorManager = ng.createKnownGeneratorManager(installedDeclaratives)
	} else {
		log.Info().Str("mode", "finalyzer").Msgf("Finalizer hook started")
		generatorManager = ng.createGenericGeneratorManager()
	}
	generatorManager.run()
}

func prepareDataFromFiles() map[string][]unstructured.Unstructured {
	log.Info().Str("type", "init").Msgf("starting prepareDataFromFiles")
	var resourceList []runtime.Object
	installedDeclaratives := make(map[string][]unstructured.Unstructured)
	files, _ := os.ReadDir(mountPath)
	err := os.Chdir(mountPath)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Can't chdir to folder with declarative files")
	}
	for _, file := range files {
		name := file.Name()

		if strings.HasPrefix(name, "..") {
			continue
		}

		yamlFile, _ := os.ReadFile(file.Name())
		log.Info().Str("type", "init").Str("name", file.Name()).Msgf("yaml file name")
		log.Info().Str("type", "init").Msgf("yaml file content:\n%s", string(yamlFile))
		resourceList = append(resourceList, GetObjects(yamlFile)...)
	}
	for _, obj := range resourceList {
		objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Declarative structure can't be transformed to Unstructured type")
		}
		declarative := unstructured.Unstructured{Object: objMap}
		resKind := declarative.GetObjectKind().GroupVersionKind().Kind
		log.Info().Str("type", "init").Str("kind", resKind).Str("name", declarative.GetName()).Msgf("transformed resource name")
		installedDeclaratives[resKind] = append(installedDeclaratives[resKind], declarative)
	}
	return installedDeclaratives
}
