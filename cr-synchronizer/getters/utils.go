package getters

import (
	"context"
	"github.com/rs/zerolog/diode"
	"os"
	"strings"

	ncapi "github.com/netcracker/cr-synchronizer/clientset"
	"github.com/rs/zerolog"
	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	mountPath                      = "/mnt/declaratives"
	componentName                  = "core-operator"
	componentInstance              = "declaration-waiter"
	coreDeclarativeEventsGenerator = "nc-operator"
	GroupName                      = "core.netcracker.com"
	GroupVersion                   = "v1"
	CdnGroupName                   = "cdn.netcracker.com"
	CdnGroupVersion                = "v1"
	manager                        = "cr-synchronizer"
)

var deploymentName, serviceName, namespace, receiverKind string
var log zerolog.Logger
var wr diode.Writer

var labels = map[string]string{
	"deployment.netcracker.com/sessionId":     os.Getenv("DEPLOYMENT_SESSION_ID"),
	"app.kubernetes.io/name":                  serviceName,
	"app.kubernetes.io/managed-by":            manager,
	"app.kubernetes.io/part-of":               os.Getenv("APPLICATION_NAME"),
	"app.kubernetes.io/processed-by-operator": componentName,
}

func setEventReceiver(ctx context.Context, clientSet *ncapi.Clientset) runtime.Object {
	var runtimeReceiver runtime.Object
	//for ArgoCd DEPLOYMENT_RESOURCE_NAME == SERVICE_NAME-v1 (if exists)
	deployment, err := clientSet.AppsV1().Deployments(namespace).Get(ctx, os.Getenv("DEPLOYMENT_RESOURCE_NAME"), k8sv1.GetOptions{})
	if err != nil {
		deployment, err = clientSet.AppsV1().Deployments(namespace).Get(ctx, os.Getenv("SERVICE_NAME"), k8sv1.GetOptions{})
	}
	if err != nil {
		job, err := clientSet.BatchV1().Jobs(namespace).Get(ctx, os.Getenv("WAIT_JOB_NAME"), k8sv1.GetOptions{})
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Cant get runtimeObject to send events")
		}
		runtimeReceiver = job
		receiverKind = "Job"
		if len(os.Getenv("DEPLOYMENT_RESOURCE_NAME")) != 0 {
			deploymentName = os.Getenv("DEPLOYMENT_RESOURCE_NAME")
			log.Warn().Msg(".Values.DEPLOYMENT_RESOURCE_NAME is used as deployment name. But it doesnt exist in previous installation")
		} else {
			deploymentName = os.Getenv("SERVICE_NAME")
			log.Warn().Msg(".Values.SERVICE_NAME is used as deployment name. But it doesnt exist in previous installation")
		}
	} else {
		runtimeReceiver = deployment
		receiverKind = "Deployment"
		deploymentName = deployment.GetName()
	}
	log.Info().Str("type", "init").Str("kind", receiverKind).Msgf("runtime object kind to receive events")
	return runtimeReceiver
}

func GetCurrentNS() string {
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns
	}
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return "default"
}

func getObjectMap(tmpl []byte) [][]byte {
	splits := strings.Split(string(tmpl), "---")
	objectList := make([][]byte, 0, len(splits))
	for _, v := range splits {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		objectList = append(objectList, []byte(v))
	}
	return objectList
}

func GetObjects(manifestData []byte) []runtime.Object {
	list := getObjectMap(manifestData)
	m := make([]runtime.Object, 0, len(list))
	for _, v := range list {
		var Codec = serializer.NewCodecFactory(scheme.Scheme).
			UniversalDecoder(scheme.Scheme.PrioritizedVersionsAllGroups()...)
		data, err := runtime.Decode(Codec, v)
		if err != nil {
			log.Fatal().Any("yaml object", string(v)).Stack().Err(err).Msg("Can't decode object scheme")
		}
		m = append(m, data)
	}
	return m
}
