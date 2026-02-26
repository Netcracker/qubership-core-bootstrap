package getters

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	v12 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	ncapi "github.com/netcracker/cr-synchronizer/clientset"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	genericRunner = "genericDeclarativeClient"
)

type GenericRunner struct {
	DeploymentGenerator
}

func NewGenericRunnerGenerator(ctx context.Context, client dynamic.Interface, recorder EventRecorder, clientset ncapi.Interface, scheme *runtime.Scheme, runtimeReceiver runtime.Object, timeoutSeconds int) *GenericRunner {
	return &GenericRunner{
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

func (ng *GenericRunner) Name() string {
	return genericRunner
}

func (ng *GenericRunner) processResourcesForLabel(schemeRes schema.GroupVersionResource, objPlural, deploymentSessionId, labelKey string) {
	log.Info().Str("type", "genericWaiter").Str("resource", schemeRes.Resource).Str("version", schemeRes.Version).Str("group", schemeRes.Group).Str(labelKey, serviceName).Str("sessionId", deploymentSessionId).Msgf("checking resource in kubernetes to wait for")
	listRes, err := ng.client.Resource(schemeRes).Namespace(namespace).List(ng.ctx, k8sv1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s, %s=%s", "deployment.netcracker.com/sessionId", deploymentSessionId, labelKey, serviceName)})
	if err != nil {
		log.Warn().Stack().Str("plurals", objPlural).Str("sessionID", deploymentSessionId).Err(err).Msg("Failed to find plurals in current session")
	}
	if listRes != nil {
		for _, declarative := range listRes.Items {
			log.Info().Str("type", "genericWaiter").Str("declarativeName", declarative.GetName()).Str("group", schemeRes.Group).Msgf("starting waiter for declarative")
			ng.GenericWaiter(schemeRes, declarative)
			log.Info().Str("plural", objPlural).Msgf("Declaratives updated")
		}
	}
}

func (ng *GenericRunner) Generate() {
	deploymentSessionId := os.Getenv("DEPLOYMENT_SESSION_ID")
	objPlurals := []string{"configurationpackages", "smartplugplugins", "meshes", "securities", "composites", "maases", "dbaases", "gateways", RouteResourceName}
	definedPl, found := os.LookupEnv("DECLARATIONS_PLURALS")
	if found && len(definedPl) > 0 {
		objPlurals = strings.Split(definedPl, ",")
	}
	for _, objPlural := range objPlurals {
		var schemeResources schema.GroupVersionResource
		if strings.EqualFold(objPlural, "cdns") {
			schemeResources = schema.GroupVersionResource{Group: CdnGroupName, Version: CdnGroupVersion, Resource: objPlural}
		} else if strings.EqualFold(objPlural, RouteResourceName) {
			schemeResources = schema.GroupVersionResource{Group:RouteGroupName, Version: RouteGroupVersion, Resource: objPlural}
		} else {
			schemeResources = schema.GroupVersionResource{Group: GroupName, Version: GroupVersion, Resource: objPlural}
		}

		ng.processResourcesForLabel(schemeResources, objPlural, deploymentSessionId, "app.kubernetes.io/name")
		ng.processResourcesForLabel(schemeResources, objPlural, deploymentSessionId, "app.kubernetes.io/instance")
	}

	ng.v1DeploymentAndHpaMigration()
}

func (ng *GenericRunner) v1DeploymentAndHpaMigration() {
	// migration if we have old v0 deployment migrated to facade v1 deployment (old must be deleted)
	log.Info().Str("type", "migration").Msgf("starting deployment version migration check")

	listDeplSet, err := ng.clientset.AppsV1().Deployments(namespace).List(ng.ctx, k8sv1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", "app.kubernetes.io/name", os.Getenv("SERVICE_NAME"))})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("error during depl list clientset")
	}

	// assuming we can have old depl v0, new facade depl v1 and some other depl, e.g. composite gateway
	var deplV0 v12.Deployment
	var deplV1 v12.Deployment
	for _, depl := range listDeplSet.Items {
		log.Info().Str("type", "migration").Str("deployment name in depl list with label 'app.kubernetes.io/name' == SERVICE_NAME ", depl.Name)
		for _, depl2 := range listDeplSet.Items {
			if depl.Name == depl2.Name+"-v1" {
				deplV1 = depl
				deplV0 = depl2
			}
		}
	}

	if deplV1.Name == "" {
		log.Info().Str("type", "migration").Msgf("no v1 deployment found, skipping deployment migration step")
		return
	}

	// need to check label, because we could have depl v1 and composite gateway named just as 'name' label
	if deplV0.Name != "" && deplV0.Labels != nil && deplV0.Labels["app.kubernetes.io/managed-by-operator"] == "facade-operator" {
		log.Info().Str("type", "migration").Msgf("v0Depl is managed by operator, skipping migration")
		return
	}

	isReady := CheckDeploymentStatus(ng.ctx, ng.clientset, namespace, deplV1.Name)
	if isReady {
		log.Info().Str("type", "migration").Msgf("deployment v1 is ready")
	} else {
		log.Fatal().Str("type", "migration").Msgf("deployment v1 is not ready after timeout")
	}

	log.Info().Str("type", "migration").Any("deployment name", deplV0.Name).Msgf("deployment v0 deletion")
	log.Info().Str("type", "migration").Any("deployment uid", deplV0.UID).Msgf("deployment v0 deletion")
	err = ng.clientset.AppsV1().Deployments(namespace).Delete(ng.ctx, deplV0.Name, k8sv1.DeleteOptions{
		GracePeriodSeconds: int64Ptr(0),
	})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("error during depl deletion")
	} else {
		log.Info().Str("type", "migration").Msgf("deployment deletion initiated")
	}
	log.Info().Str("type", "migration").Msgf("after deployment v0 deletion")

	log.Info().Str("type", "migration").Msgf("before getting hpa")

	var schemeRes = schema.GroupVersionResource{
		Group:    "autoscaling",
		Version:  "v2",
		Resource: "horizontalpodautoscalers",
	}

	hpa, err := ng.client.Resource(schemeRes).Namespace(namespace).Get(ng.ctx, os.Getenv("SERVICE_NAME"), k8sv1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Info().Str("type", "migration").Msgf("no hpa found, migration finished")
			return
		}
		log.Fatal().Stack().Err(err).Msg("error during listing hpa")
	}

	//just in case
	if hpa == nil {
		log.Info().Str("type", "migration").Msgf("no hpa found, migration finished")
		return
	}

	log.Info().Str("type", "migration").Any("hpa", hpa).Msgf("hpa")
	log.Info().Str("type", "migration").Msgf("after getting hpa")

	log.Info().Str("type", "migration").Msgf("before deleting hpa")
	err = ng.client.Resource(schemeRes).Namespace(namespace).Delete(ng.ctx, os.Getenv("SERVICE_NAME"), k8sv1.DeleteOptions{
		GracePeriodSeconds: int64Ptr(0),
	})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("error during deleteHpa")
	}
	log.Info().Str("type", "migration").Msgf("after deleting hpa")

	_, err = ng.client.Resource(schemeRes).Namespace(namespace).Get(ng.ctx, os.Getenv("SERVICE_NAME"), k8sv1.GetOptions{})
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			log.Fatal().Stack().Err(err).Msg("error during checking deleted hpa")
		}
	}
	log.Info().Str("type", "migration").Msgf("migration finished successfully")
}

func CheckDeploymentStatus(ctx context.Context, clientset ncapi.Interface, namespace, deploymentName string) bool {
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	interval := 10 * time.Second
	attempt := 0
	for {
		select {
		case <-waitCtx.Done():
			return false
		default:
			v1Depl, err := clientset.AppsV1().Deployments(namespace).Get(waitCtx, deploymentName, k8sv1.GetOptions{})
			if err != nil {
				log.Fatal().Str("type", "migration").Stack().Err(err).Msgf("error fetching deployment")
			}

			for _, condition := range v1Depl.Status.Conditions {
				if condition.Type == v12.DeploymentAvailable && condition.Status == "True" {
					return true
				}
			}
			log.Info().Str("type", "migration").Any("v1Depl status", v1Depl.Status).Msgf("v1Depl status")

			attempt++
			log.Info().Str("type", "migration").Int("attempt", attempt).Msgf("deployment is not ready yet ")
			time.Sleep(interval)
		}
	}
}

func int64Ptr(i int64) *int64 { return &i }
