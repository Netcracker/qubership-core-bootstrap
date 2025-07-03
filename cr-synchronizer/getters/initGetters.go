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
	"strings"
	"sync"
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
	var generatorsWaitGroup sync.WaitGroup
	generatorsWaitGroup.Add(len(gm.generators))
	for name, generator := range gm.generators {
		go func() {
			defer generatorsWaitGroup.Done()
			log.Info().Str("type", "generator").Str("name", name).Msgf("Register new waiter")
			generator.Generate()
		}()
	}
	generatorsWaitGroup.Wait()
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
		v1Core.EventSource{Component: coreDeclarativeEventsGenerator, Host: componentInstance})
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Dynamic Client can't be initialized")
	}
	NewDeploymentGenerator(client, recorder, clientSet, scheme.Scheme, runtimeReceiver).Run()

	log.Info().Str("type", "init").Msgf("generator finished")
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
