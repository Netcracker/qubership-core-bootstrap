package getters

import (
	"context"
	"encoding/json"
	"fmt"
	ncapi "github.com/netcracker/cr-synchronizer/clientset"
	v1Core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"os"
	"time"
)

const (
	MaaSKind  = "MaaS"
	DBaaSKind = "DBaaS"
)

type DeploymentGenerator struct {
	client          dynamic.Interface
	recorder        EventRecorder
	clientset       ncapi.Interface
	scheme          *runtime.Scheme
	runtimeReceiver runtime.Object
	timeoutSeconds  int
	postDeploy      bool
}

func NewDeploymentGenerator(client dynamic.Interface, recorder EventRecorder, clientset ncapi.Interface, scheme *runtime.Scheme, runtimeReceiver runtime.Object, postDeploy bool, timeoutSeconds int) *DeploymentGenerator {
	return &DeploymentGenerator{
		client:          client,
		recorder:        recorder,
		clientset:       clientset,
		scheme:          scheme,
		runtimeReceiver: runtimeReceiver,
		timeoutSeconds:  timeoutSeconds,
		postDeploy:      postDeploy,
	}
}

func (ng *DeploymentGenerator) Run() {
	var generatorManager *GeneratorManager
	if !ng.postDeploy {
		log.Info().Str("mode", "synchronizer").Msgf("Synchronizer hook started")
		installedDeclaratives := prepareDataFromFiles()
		generatorManager = ng.createKnownGeneratorManager(installedDeclaratives)
	} else {
		log.Info().Str("mode", "finalyzer").Msgf("Finalizer hook started")
		generatorManager = ng.createGenericGeneratorManager()
	}
	generatorManager.run()
}

func (ng *DeploymentGenerator) createGenericGeneratorManager() *GeneratorManager {
	generatorManager = &GeneratorManager{
		generators: make(map[string]Generator),
	}
	generatorManager.register(NewGenericRunnerGenerator(ng.client, ng.recorder, ng.clientset, ng.scheme, ng.runtimeReceiver, ng.timeoutSeconds))
	return generatorManager
}

func (ng *DeploymentGenerator) createKnownGeneratorManager(dcl map[string][]unstructured.Unstructured) *GeneratorManager {
	generatorManager = &GeneratorManager{
		generators: make(map[string]Generator),
	}
	generatorManager.register(NewMaaSesRunnerGenerator(dcl[MaaSKind], ng.client, ng.recorder, ng.clientset, ng.scheme, ng.runtimeReceiver, ng.timeoutSeconds))
	generatorManager.register(NewDBaaSesRunnerGenerator(dcl[DBaaSKind], ng.client, ng.recorder, ng.clientset, ng.scheme, ng.runtimeReceiver, ng.timeoutSeconds))
	return generatorManager
}

func (ng *DeploymentGenerator) sendEvent(iReason, iMessage, declarativeName, kindDec string) {
	podName, _ := os.Hostname()
	pod, err := ng.clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, v1.GetOptions{})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Can't get pod in current namespace")
	}

	var ownerName, ownerKind string
	var uid types.UID
	if len(pod.OwnerReferences) != 0 {
		switch pod.OwnerReferences[0].Kind {
		case "ReplicaSet":
			replica, repErr := ng.clientset.AppsV1().ReplicaSets(pod.Namespace).Get(context.TODO(), pod.OwnerReferences[0].Name, v1.GetOptions{})
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

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ng.runtimeReceiver)
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

	ng.recorder.LabeledEventf(ng.runtimeReceiver, labels, annotations, v1Core.EventTypeWarning, iReason, iMessage)

	time.Sleep(2 * time.Second)
}

func (ng *DeploymentGenerator) declarationCreator(resourceList []unstructured.Unstructured, objPlural string) (schema.GroupVersionResource, []string) {
	deploymentRes := schema.GroupVersionResource{Group: GroupName, Version: GroupVersion, Resource: objPlural}
	log.Info().Str("type", "creator").Str("group", deploymentRes.Group).Str("resource", deploymentRes.Resource).Str("version", deploymentRes.Version).Msgf("Starting to process resources")
	var resourceNames []string
	for _, declarative := range resourceList {
		jsonData, err := json.Marshal(declarative.Object)
		log.Info().Str("type", "creator").Str("name", declarative.GetName()).Str("declarative", string(jsonData)).Msgf("Starting to process single resource")

		customLabels := declarative.GetLabels()
		customLabels["app.kubernetes.io/managed-by"] = manager
		declarative.SetLabels(customLabels)
		resourceNames = append(resourceNames, declarative.GetName())
		priorDeclarative, err := ng.client.Resource(deploymentRes).Namespace(namespace).Get(context.TODO(), declarative.GetName(), v1.GetOptions{})
		if err != nil {
			resp, err := ng.client.Resource(deploymentRes).Namespace(namespace).Create(context.TODO(), &declarative, v1.CreateOptions{FieldManager: "pre-hook"})
			if err != nil {
				log.Fatal().Stack().Str("name", declarative.GetName()).Err(err).Msg("Failed to create resource")
			}
			log.Info().Str("type", "creator").Str("name", resp.GetName()).Msgf("Resource had been created")
		} else {
			log.Info().Str("type", "updater").Str("name", priorDeclarative.GetName()).Msgf("priorDeclarative: %+v", priorDeclarative)
			log.Info().Str("type", "updater").Str("resourceVersion-new", declarative.GetResourceVersion()).Str("resourceVersion-old", priorDeclarative.GetResourceVersion())

			declarative.SetResourceVersion(priorDeclarative.GetResourceVersion())
			result, err := ng.client.Resource(deploymentRes).Namespace(namespace).Update(context.TODO(), &declarative, v1.UpdateOptions{FieldManager: "pre-hook"})
			if err != nil {
				log.Fatal().Stack().Str("name", declarative.GetName()).Err(err).Msg("Failed to apply resource")
			}
			log.Info().Str("type", "updater").Str("name", result.GetName()).Msgf("Resource had been applied")
		}
	}
	return deploymentRes, resourceNames
}

func (ng *DeploymentGenerator) setOwnerRef(resourceType schema.GroupVersionResource, resourceName string) {
	result, err := ng.client.Resource(resourceType).Namespace(namespace).Get(context.TODO(), resourceName, v1.GetOptions{})
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
	deployment, err := deplClient.Get(context.TODO(), deploymentName, v1.GetOptions{})
	if err != nil {
		log.Warn().Str("type", "waiter").Str("deploymentName", deploymentName).Err(err).Msg("Cant get deployment for current CR, skip owner ref update")
		return
	}

	deploymentUuid := deployment.ObjectMeta.UID

	log.Info().Str("type", "waiter").Str("deploymentName", deploymentName).Msgf("Deployment name retrieved")
	log.Info().Str("type", "waiter").Str("deploymentUuid", string(deploymentUuid)).Msgf("Deployment uid from transformed object")
	ownerRefList := make([]v1.OwnerReference, 0)
	ownerRef := &v1.OwnerReference{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Name:       deploymentName,
		UID:        deploymentUuid,
	}
	ownerRefList = append(ownerRefList, *ownerRef)

	ok := false
	for retry := 0; retry < 10; retry++ {
		// getting updated resource
		result, err := ng.client.Resource(resourceType).Namespace(namespace).Get(context.TODO(), resourceName, v1.GetOptions{})
		if err != nil {
			log.Fatal().Stack().Str("name", resourceName).Err(err).Msg("Failed to get current custom resource")
		}
		jsonData, _ := json.Marshal(result.Object)
		log.Info().Str("type", "waiter").Str("name", result.GetName()).Str("received resource", string(jsonData)).Msgf("setting owner ref for resource")

		result.SetOwnerReferences(ownerRefList)
		updatedResult, err := ng.client.Resource(resourceType).Namespace(namespace).Update(context.TODO(), result, v1.UpdateOptions{FieldManager: "pre-hook"})
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

func (ng *DeploymentGenerator) handlePhaseChange(resourceType schema.GroupVersionResource, resourceName string, result *unstructured.Unstructured, phaseField string) bool {
	kind := result.GetKind()
	switch phaseField {
	case "WaitingForDependency", "BackingOff", "Updating":
		log.Info().Str("type", "waiter").Str("name", resourceName).Str("kind", kind).Str("phase", phaseField).Msgf("Declarative not ready")
		// Wait for next event
		return false
	case "InvalidConfiguration":
		cReason, isFound, err := unstructured.NestedString(result.Object, "status", "phase")
		if !isFound || err != nil {
			log.Fatal().Stack().Str("name", resourceName).Str("kind", kind).Err(err).Msg("Can't find or parse reason field")
		}
		cMessage, isFound, err := unstructured.NestedString(result.Object, "status", "phase")
		if !isFound || err != nil {
			log.Fatal().Stack().Str("name", resourceName).Str("kind", kind).Err(err).Msg("Can't find or parse message field")
		}
		ng.sendEvent(cReason, cMessage, resourceName, kind)
		log.Fatal().Stack().Str("name", resourceName).Str("kind", kind).Str("phase", phaseField).Msgf(cReason)
		return false
	case "Updated":
		log.Info().Str("type", "waiter").Str("name", resourceName).Msg("start setting owner reference on stable phase 'Updated'")
		ng.setOwnerRef(resourceType, resourceName)
		log.Info().Str("type", "waiter").Str("name", resourceName).Msg("finished setting owner reference on stable phase 'Updated'")
		return true
	default:
		log.Info().Str("type", "waiter").Str("name", resourceName).Str("kind", kind).Msgf("Resource still does not have phase field")
		return false
	}
}

func (ng *DeploymentGenerator) declarationWaiter(resourceType schema.GroupVersionResource, resourceName string) {
	log.Info().Str("type", "waiter").Str("name", resourceName).Str("resourceGroup", resourceType.Group).Msgf("starting waiter for resource")

	watcher, err := ng.client.Resource(resourceType).Namespace(namespace).Watch(context.TODO(), v1.ListOptions{
		FieldSelector:  "metadata.name=" + resourceName,
		TimeoutSeconds: func() *int64 { t := int64(ng.timeoutSeconds); return &t }(),
	})
	if err != nil {
		log.Fatal().Stack().Str("name", resourceName).Str("group", resourceType.Group).Err(err).Msg("Failed to start watch on declaration")
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			log.Warn().Str("name", resourceName).Msg("Received non-unstructured object from watch")
			return
		}
		phaseField, isFound, err := unstructured.NestedString(obj.Object, "spec", "status", "phase")
		if !isFound {
			log.Warn().Str("type", "waiter").Stack().Str("name", resourceName).Str("group", resourceType.Group).Err(err).Msg("Phase field not found")
		}
		if err != nil {
			log.Warn().Str("type", "waiter").Stack().Str("name", resourceName).Str("group", resourceType.Group).Err(err).Msg("Phase field lookup error")
		}
		if ng.handlePhaseChange(resourceType, resourceName, obj, phaseField) {
			return
		}
	}
	log.Info().Str("type", "waiter").Str("resource", resourceName).Str("group", resourceType.Group).Msgf("Waiting done")
	log.Info().Str("type", "waiter").Str("name", resourceName).Str("resource", resourceType.Resource).Msgf("finished waiter for resource")
}

func (ng *DeploymentGenerator) GenericWaiter(deploymentRes schema.GroupVersionResource, declarativeAsUnstructured unstructured.Unstructured) {
	to := time.After(time.Duration(ng.timeoutSeconds) * time.Second)
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
				declarativeAsUnstructured, err := ng.client.Resource(deploymentRes).Namespace(namespace).Get(context.TODO(), declarativeAsUnstructured.GetName(), v1.GetOptions{})
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
