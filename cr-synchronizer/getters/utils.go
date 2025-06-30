package getters

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/netcracker/cr-synchronizer/api/types/v1"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/diode"

	ncapi "github.com/netcracker/cr-synchronizer/clientset"
	"github.com/rs/zerolog"
	v1Core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	mountPath                      = "/Users/estetsenko/declaratives"
	MaaSKind                       = "MaaS"
	MeshKind                       = "Mesh"
	DBaaSKind                      = "DBaaS"
	SecurityKind                   = "Security"
	CompositeKind                  = "Composite"
	component_name                 = "core-operator"
	component_instance             = "declaration-waiter"
	coreDeclarativeEventsGenerator = "nc-operator"
	deploymentGenerator            = "deploymentEventsGenerator"
	manager                        = "cr-synchronizer"
)

var allCrCheckersWaitGroup sync.WaitGroup

var rptm int
var deploymentName, serviceName, namespace, receiverKind string
var postdeploy bool
var log zerolog.Logger
var wr diode.Writer

var labels = map[string]string{
	"deployment.qubership.org/sessionId":      os.Getenv("DEPLOYMENT_SESSION_ID"),
	"app.kubernetes.io/name":                  serviceName,
	"app.kubernetes.io/managed-by":            manager,
	"app.kubernetes.io/part-of":               os.Getenv("APPLICATION_NAME"),
	"app.kubernetes.io/processed-by-operator": component_name,
}

func setEventReceiver(clientSet *ncapi.Clientset) runtime.Object {
	var runtimeReceiver runtime.Object
	//for ArgoCd DEPLOYMENT_RESOURCE_NAME == SERVICE_NAME-v1 (if exists)
	deployment, err := clientSet.AppsV1().Deployments(namespace).Get(context.TODO(), os.Getenv("DEPLOYMENT_RESOURCE_NAME"), k8sv1.GetOptions{})
	if err != nil {
		deployment, err = clientSet.AppsV1().Deployments(namespace).Get(context.TODO(), os.Getenv("SERVICE_NAME"), k8sv1.GetOptions{})
	}
	if err != nil {
		job, err := clientSet.BatchV1().Jobs(namespace).Get(context.TODO(), os.Getenv("WAIT_JOB_NAME"), k8sv1.GetOptions{})
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

func GoCrChecker(f func()) {
	allCrCheckersWaitGroup.Add(1)
	go func() {
		defer allCrCheckersWaitGroup.Done()
		f()
	}()
}

func (c *DeploymentGenerator) sendEvent(iReason, iMessage, declarativeName, kindDec string) {
	podName, _ := os.Hostname()
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, k8sv1.GetOptions{})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Can't get pod in current namespace")
	}

	var ownerName, ownerKind string
	var uid types.UID
	if len(pod.OwnerReferences) != 0 {
		switch pod.OwnerReferences[0].Kind {
		case "ReplicaSet":
			replica, repErr := c.clientset.AppsV1().ReplicaSets(pod.Namespace).Get(context.TODO(), pod.OwnerReferences[0].Name, k8sv1.GetOptions{})
			if repErr != nil {
				log.Fatal().Stack().Err(err).Msg("Can't get replicas in current namespace")
			}
			ownerName = replica.OwnerReferences[0].Name
			uid = replica.OwnerReferences[0].UID
			ownerKind = "Deployment"
		case "DaemonSet", "StatefulSet":
			ownerName = pod.OwnerReferences[0].Name
			ownerKind = pod.OwnerReferences[0].Kind
			uid = pod.OwnerReferences[0].UID
		case "Job":
			ownerName = pod.OwnerReferences[0].Name
			ownerKind = pod.OwnerReferences[0].Kind
			uid = pod.OwnerReferences[0].UID
		default:
			log.Warn().Str("kind", pod.OwnerReferences[0].Kind).Msgf("Could not find resource manager")
		}
	}

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(c.runtimeReceiver)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("RuntimeObject can't be transformed to Unstructured")
	}
	receiver := unstructured.Unstructured{Object: unstructuredObj}

	annotations := map[string]string{
		"relatedCR":                    fmt.Sprintf("%s/%s", kindDec, declarativeName),
		"producedByEntity":             fmt.Sprintf("%s/%s", ownerKind, ownerName),
		"producerUID":                  string(uid),
		"producedByPod":                podName,
		"relatedToRuntimeObject":       fmt.Sprintf("%s/%s", receiverKind, receiver.GetName()),
		"runtimeObjectResourceVersion": receiver.GetResourceVersion(),
	}

	c.recorder.LabeledEventf(c.runtimeReceiver, labels, annotations, v1Core.EventTypeWarning, iReason, iMessage)

	time.Sleep(2 * time.Second)
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

func (ng *DeploymentGenerator) declarationCreator(resourceList []unstructured.Unstructured, objPlural string) map[schema.GroupVersionResource][]string {
	deploymentResources := make(map[schema.GroupVersionResource][]string)
	for _, groupName := range v1.CoreApiGroupNames {
		deploymentRes := schema.GroupVersionResource{Group: groupName, Version: v1.GroupVersion, Resource: objPlural}
		log.Info().Str("type", "creator").Str("group", deploymentRes.Group).Str("resource", deploymentRes.Resource).Str("version", deploymentRes.Version).Msgf("Starting to process resources")
		deploymentResources[deploymentRes] = make([]string, 0)
		for _, declarative := range resourceList {
			jsonData, err := json.Marshal(declarative.Object)
			log.Info().Str("type", "creator").Str("name", declarative.GetName()).Str("declarative", string(jsonData)).Msgf("Starting to process single resource")

			customLabels := declarative.GetLabels()
			customLabels["app.kubernetes.io/managed-by"] = manager
			declarative.SetLabels(customLabels)
			deploymentResources[deploymentRes] = append(deploymentResources[deploymentRes], declarative.GetName())
			priorDeclarative, err := ng.client.Resource(deploymentRes).Namespace(namespace).Get(context.TODO(), declarative.GetName(), k8sv1.GetOptions{})
			if err != nil {
				resp, err := ng.client.Resource(deploymentRes).Namespace(namespace).Create(context.TODO(), &declarative, k8sv1.CreateOptions{FieldManager: "pre-hook"})
				if err != nil {
					log.Fatal().Stack().Str("name", declarative.GetName()).Err(err).Msg("Failed to create resource")
				}
				log.Info().Str("type", "creator").Str("name", resp.GetName()).Msgf("Resource had been created")
			} else {
				log.Info().Str("type", "updater").Str("name", priorDeclarative.GetName()).Msgf("priorDeclarative: %+v", priorDeclarative)
				log.Info().Str("type", "updater").Str("resourceVersion-new", declarative.GetResourceVersion()).Str("resourceVersion-old", priorDeclarative.GetResourceVersion())

				declarative.SetResourceVersion(priorDeclarative.GetResourceVersion())
				result, err := ng.client.Resource(deploymentRes).Namespace(namespace).Update(context.TODO(), &declarative, k8sv1.UpdateOptions{FieldManager: "pre-hook"})
				if err != nil {
					log.Fatal().Stack().Str("name", declarative.GetName()).Err(err).Msg("Failed to apply resource")
				}
				log.Info().Str("type", "updater").Str("name", result.GetName()).Msgf("Resource had been applied")
			}
		}
	}

	return deploymentResources
}

func (ng *DeploymentGenerator) setOwnerRef(resourceType schema.GroupVersionResource, resourceName string) {
	result, err := ng.client.Resource(resourceType).Namespace(namespace).Get(context.TODO(), resourceName, k8sv1.GetOptions{})
	if err != nil {
		log.Fatal().Stack().Str("name", resourceName).Err(err).Msg("Failed to get current custom resource")
	}
	jsonData, _ := json.Marshal(result.Object)
	log.Info().Str("type", "waiter").Str("name", result.GetName()).Str("resource", string(jsonData)).Msgf("requested resource for owner ref")

	if result.GetOwnerReferences() != nil {
		log.Info().Str("type", "waiter").Any("resourceName", resourceName).Msgf("owner reference is not nil, skipping setting")
		return
	}

	deplClient := ng.clientset.AppsV1().Deployments(namespace)
	deployment, err := deplClient.Get(context.TODO(), deploymentName, k8sv1.GetOptions{})
	if err != nil {
		log.Warn().Str("type", "waiter").Str("deploymentName", deploymentName).Err(err).Msg("Cant get deployment for current CR, skip owner ref update")
		return
	}

	deploymentUuid := deployment.ObjectMeta.UID

	log.Info().Str("type", "waiter").Str("deploymentName", deploymentName).Msgf("Deployment name retrieved")
	log.Info().Str("type", "waiter").Str("deploymentUuid", string(deploymentUuid)).Msgf("Deployment uid from transformed object")
	ownerRefList := make([]k8sv1.OwnerReference, 0)
	ownerRef := &k8sv1.OwnerReference{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Name:       deploymentName,
		UID:        deploymentUuid,
	}
	ownerRefList = append(ownerRefList, *ownerRef)

	ok := false
	for retry := 0; retry < 10; retry++ {
		// getting updated resource
		result, err := ng.client.Resource(resourceType).Namespace(namespace).Get(context.TODO(), resourceName, k8sv1.GetOptions{})
		if err != nil {
			log.Fatal().Stack().Str("name", resourceName).Err(err).Msg("Failed to get current custom resource")
		}
		jsonData, _ := json.Marshal(result.Object)
		log.Info().Str("type", "waiter").Str("name", result.GetName()).Str("received resource", string(jsonData)).Msgf("setting owner ref for resource")

		result.SetOwnerReferences(ownerRefList)
		updatedResult, err := ng.client.Resource(resourceType).Namespace(namespace).Update(context.TODO(), result, k8sv1.UpdateOptions{FieldManager: "pre-hook"})
		if err != nil {
			log.Warn().Str("type", "waiter").Str("err", err.Error()).Msgf("error from kubernetes after update")
			if k8serrors.IsConflict(err) {
				log.Warn().Str("type", "waiter").Str("name", resourceName).Msg("Conflict detected during owner reference update, retrying...")
				time.Sleep(5 * time.Second)
				continue
			}
			log.Fatal().Stack().Str("name", resourceName).Err(err).Msg("Failed to update owner reference")
		}

		jsonData, _ = json.Marshal(updatedResult.Object)
		log.Info().Str("type", "waiter").Str("name", result.GetName()).Str("updated resource with owner ref", string(jsonData)).Msgf("updated res")

		ok = true
		break
	}

	if !ok {
		log.Fatal().Stack().Str("name", resourceName).Err(err).Msg("Failed to update reference after retries")
	}

	log.Info().Str("type", "waiter").Str("resourceName", resourceName).Msgf("Owner reference updated")

}

func (ng *DeploymentGenerator) declarationWaiter(resourceType schema.GroupVersionResource, resourceName string) {
	log.Info().Str("type", "waiter").Str("name", resourceName).Str("resourceGroup", resourceType.Group).Msgf("starting waiter for resource")

	to := time.After(time.Duration(rptm) * time.Second)
	done := make(chan bool)
	var result *unstructured.Unstructured
	var err error
	go func() {
		defer log.Info().Str("type", "waiter").Str("resource", resourceName).Str("group", resourceType.Group).Msgf("Waiting done")
		for {
			select {
			case <-done:
				log.Printf("done return")
				return
			case <-to:
				ng.sendEvent("TimeOutReached", "Declaratives failed to progress", resourceName, result.GetKind())
				log.Fatal().Stack().Str("name", resourceName).Str("kind", result.GetKind()).Err(err).Msg("TimeOutReached")
			default:
				result, err = ng.client.Resource(resourceType).Namespace(namespace).Get(context.TODO(), resourceName, k8sv1.GetOptions{})
				if err != nil {
					log.Fatal().Stack().Str("name", resourceName).Str("group", resourceType.Group).Err(err).Msg("Failed to get declaration")
				}
				phaseField, isFound, err := unstructured.NestedString(result.Object, "status", "phase")
				if !isFound {
					log.Warn().Str("type", "waiter").Stack().Str("name", resourceName).Str("group", resourceType.Group).Err(err).Msg("Phase field not found")
				}
				if err != nil {
					log.Warn().Str("type", "waiter").Stack().Str("name", resourceName).Str("group", resourceType.Group).Err(err).Msg("Phase field lookup error")
				}

				switch phaseField {
				case "WaitingForDependency":
					log.Info().Str("type", "waiter").Str("name", resourceName).Str("kind", result.GetKind()).Str("phase", phaseField).Msgf("Declarative not ready")
					time.Sleep(5 * time.Second)
				case "BackingOff":
					log.Info().Str("type", "waiter").Str("name", resourceName).Str("kind", result.GetKind()).Str("phase", phaseField).Msgf("Declarative not ready")
					time.Sleep(5 * time.Second)
				case "Updating":
					log.Info().Str("type", "waiter").Str("name", resourceName).Str("kind", result.GetKind()).Str("phase", phaseField).Msgf("Declarative not ready")
					time.Sleep(5 * time.Second)
				case "InvalidConfiguration":
					cReason, isFound, err := unstructured.NestedString(result.Object, "status", "phase")
					if !isFound {
						log.Fatal().Stack().Str("name", resourceName).Str("kind", result.GetKind()).Err(err).Msg("Cant find reason field")
					}
					if err != nil {
						log.Fatal().Stack().Str("name", resourceName).Str("kind", result.GetKind()).Err(err).Msg("Error searching reason field")
					}
					cMessage, isFound, err := unstructured.NestedString(result.Object, "status", "phase")
					if !isFound {
						log.Fatal().Stack().Str("name", resourceName).Str("kind", result.GetKind()).Err(err).Msg("Cant find message field")
					}
					if err != nil {
						log.Fatal().Stack().Str("name", resourceName).Str("kind", result.GetKind()).Err(err).Msg("Error searching message field")
					}
					ng.sendEvent(cReason, cMessage, resourceName, result.GetKind())
					log.Fatal().Stack().Str("name", resourceName).Str("kind", result.GetKind()).Str("phase", phaseField).Msgf(cReason)
				case "Updated":
					log.Info().Str("type", "waiter").Str("name", resourceName).Msg("start setting owner reference on stable phase 'Updated'")
					ng.setOwnerRef(resourceType, resourceName)
					log.Info().Str("type", "waiter").Str("name", resourceName).Msg("finished setting owner reference on stable phase 'Updated'")
					done <- true
					return
				default:
					log.Info().Str("type", "waiter").Str("name", resourceName).Str("kind", result.GetKind()).Msgf("Resource still not have phase field")
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
	<-done

	log.Info().Str("type", "waiter").Str("name", resourceName).Str("resource", resourceType.Resource).Msgf("finished waiter for resource")
}

func (ng *DeploymentGenerator) GenericWaiter(deploymentRes schema.GroupVersionResource, declarativeAsUnstructured unstructured.Unstructured) {
	to := time.After(time.Duration(rptm) * time.Second)
	done := make(chan bool)
	var err error
	go func() {
		defer log.Info().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Str("group", deploymentRes.Group).Msgf("Waiting done")
		for {
			select {
			case <-done:
				return
			case <-to:
				ng.sendEvent("TimeOutReached", "Declaratives failed to progress", declarativeAsUnstructured.GetName(), declarativeAsUnstructured.GetKind())
				log.Fatal().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Err(err).Msg("TimeOutReached")
			default:
				declarativeAsUnstructured, err := ng.client.Resource(deploymentRes).Namespace(namespace).Get(context.TODO(), declarativeAsUnstructured.GetName(), k8sv1.GetOptions{})
				if err != nil {
					log.Warn().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("group", deploymentRes.Group).Err(err).Msg("Resource cant be fetched")
				}
				phaseField, isFound, err := unstructured.NestedString(declarativeAsUnstructured.Object, "status", "phase")
				if !isFound {
					log.Warn().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("group", deploymentRes.Group).Err(err).Msg("Phase field not found")
				}
				if err != nil {
					log.Warn().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("group", deploymentRes.Group).Err(err).Msg("Phase field lookup error")
				}
				switch phaseField {
				case "WaitingForDependency":
					log.Info().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Str("phase", phaseField).Msgf("Declarative not ready")
					time.Sleep(5 * time.Second)
				case "BackingOff":
					log.Info().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Str("phase", phaseField).Msgf("Declarative not ready")
					time.Sleep(5 * time.Second)
				case "Updating":
					log.Info().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Str("phase", phaseField).Msgf("Declarative not ready")
					time.Sleep(5 * time.Second)
				case "InvalidConfiguration":
					cReason, isFound, err := unstructured.NestedString(declarativeAsUnstructured.Object, "status", "phase")
					if !isFound {
						log.Fatal().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Err(err).Msg("Cant find reason field")
					}
					if err != nil {
						log.Fatal().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Err(err).Msg("Error searching reason field")
					}
					cMessage, isFound, err := unstructured.NestedString(declarativeAsUnstructured.Object, "status", "phase")
					if !isFound {
						log.Fatal().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Err(err).Msg("Cant find message field")
					}
					if err != nil {
						log.Fatal().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Err(err).Msg("Error searching message field")
					}
					ng.sendEvent(cReason, cMessage, declarativeAsUnstructured.GetName(), declarativeAsUnstructured.GetKind())
					log.Fatal().Stack().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Str("phase", phaseField).Msgf(cReason)
				case "Updated":
					log.Info().Str("type", "waiter").Str("name", declarativeAsUnstructured.GetName()).Msg("start setting owner reference on stable phase 'Updated'")
					ng.setOwnerRef(deploymentRes, declarativeAsUnstructured.GetName())
					log.Info().Str("type", "waiter").Str("name", declarativeAsUnstructured.GetName()).Msg("finished setting owner reference on stable phase 'Updated'")
					done <- true
					return
				default:
					log.Info().Str("name", declarativeAsUnstructured.GetName()).Str("kind", declarativeAsUnstructured.GetKind()).Str("phase", phaseField).Msgf("Resource still not have phase field")
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
	<-done
}
