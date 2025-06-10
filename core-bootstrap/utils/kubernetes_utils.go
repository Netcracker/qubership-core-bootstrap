package utils

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strings"
)

type K8sVersionedResourceType int

const (
	DeploymentAppsV1 K8sVersionedResourceType = iota
	ServiceV1
	ConfigMapV1
	HorizontalPodAutoscalerAutoscalingV2
	HorizontalPodAutoscalerAutoscalingV2Beta2
	PodMonitorMonitoringCoreosComV1
)

func (resourceType K8sVersionedResourceType) Kind() string {
	switch resourceType {
	case DeploymentAppsV1:
		return "Deployment"
	case ServiceV1:
		return "Service"
	case ConfigMapV1:
		return "ConfigMap"
	case HorizontalPodAutoscalerAutoscalingV2, HorizontalPodAutoscalerAutoscalingV2Beta2:
		return "HorizontalPodAutoscaler"
	case PodMonitorMonitoringCoreosComV1:
		return "PodMonitor"
	default:
		return fmt.Sprintf("<unknown Kind of K8sVersionedResourceType: %v>", int(resourceType))
	}
}

func (resourceType K8sVersionedResourceType) AppVersion() string {
	switch resourceType {
	case DeploymentAppsV1:
		return "apps/v1"
	case ServiceV1, ConfigMapV1:
		return "v1"
	case HorizontalPodAutoscalerAutoscalingV2:
		return "autoscaling/v2"
	case HorizontalPodAutoscalerAutoscalingV2Beta2:
		return "autoscaling/v2beta2"
	case PodMonitorMonitoringCoreosComV1:
		return "monitoring.coreos.com/v1"
	default:
		return fmt.Sprintf("<unknown AppVersion of K8sVersionedResourceType: %v>", int(resourceType))
	}
}

var (
	K8sClient        *kubernetes.Clientset
	K8sDynamicClient *dynamic.DynamicClient
)

func init() {
	var err error
	K8sClient, err = newKubernetesClient()
	if err != nil {
		logger.Panic("error creating kubernetes client: %v", err)
	}
	K8sDynamicClient, err = newKubernetesDynamicClient()
	if err != nil {
		logger.Panic("error creating kubernetes dynamic client: %v", err)
	}
}

func newKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}
	return kubernetes.NewForConfig(config)
}

func newKubernetesDynamicClient() (*dynamic.DynamicClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}
	return dynamic.NewForConfig(config)
}

func GetExistingSecret(ctx context.Context, namespace string, secretName string) (*v1.Secret, error) {
	secret, err := K8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}

	return secret, nil
}

func CreateOrUpdateSecret(ctx context.Context, namespace string, secret *v1.Secret) error {
	_, err := K8sClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			_, updateErr := K8sClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
			return updateErr
		}
		return err
	}
	return nil
}

func CreateSecretWithDbCredsData(ctx context.Context, namespace string, secretName string, data map[string][]byte) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/part-of":    "Cloud-Core",
				"app.kubernetes.io/managed-by": "saasDeployer",
			},
		},
		Data: data,
		Type: "Opaque",
	}

	err := CreateOrUpdateSecret(ctx, namespace, secret)
	if err != nil {
		return LogError(logger, ctx, "Error creating db secret: %v", err)
	}

	logger.InfoC(ctx, fmt.Sprintf("Secret %s created successfully", secretName))
	return nil
}

func DeleteK8sDeployment(ctx context.Context, namespace string, deploymentName string) error {
	return deleteK8sResource(ctx, DeploymentAppsV1, namespace, deploymentName, func(ctx context.Context, k8sResourceName string) error {
		return K8sClient.AppsV1().Deployments(namespace).Delete(ctx, deploymentName, metav1.DeleteOptions{})
	})
}

func DeleteK8sService(ctx context.Context, namespace string, serviceName string) error {
	return deleteK8sResource(ctx, ServiceV1, namespace, serviceName, func(ctx context.Context, k8sResourceName string) error {
		return K8sClient.CoreV1().Services(namespace).Delete(ctx, serviceName, metav1.DeleteOptions{})
	})
}

func DeleteK8sConfigMap(ctx context.Context, namespace string, configMapName string) error {
	return deleteK8sResource(ctx, ConfigMapV1, namespace, configMapName, func(ctx context.Context, k8sResourceName string) error {
		return K8sClient.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
	})
}

func DeleteK8sHorizontalPodAutoscalerV2(ctx context.Context, namespace string, horizontalPodAutoscalerName string) error {
	return deleteK8sResource(ctx, HorizontalPodAutoscalerAutoscalingV2, namespace, horizontalPodAutoscalerName, func(ctx context.Context, k8sResourceName string) error {
		return K8sClient.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, horizontalPodAutoscalerName, metav1.DeleteOptions{})
	})
}

func DeleteK8sHorizontalPodAutoscalerV2Beta2(ctx context.Context, namespace string, horizontalPodAutoscalerName string) error {
	return deleteK8sResource(ctx, HorizontalPodAutoscalerAutoscalingV2Beta2, namespace, horizontalPodAutoscalerName, func(ctx context.Context, k8sResourceName string) error {
		return K8sClient.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).Delete(ctx, horizontalPodAutoscalerName, metav1.DeleteOptions{})
	})
}

func DeleteK8sPodMonitor(ctx context.Context, namespace string, podMonitorName string) error {
	return deleteK8sResource(ctx, PodMonitorMonitoringCoreosComV1, namespace, podMonitorName, func(ctx context.Context, k8sResourceName string) error {
		gvr := schema.GroupVersionResource{
			Group:    "monitoring.coreos.com",
			Version:  "v1",
			Resource: "podmonitors",
		}
		return K8sDynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, podMonitorName, metav1.DeleteOptions{})
	})
}

func deleteK8sResource(ctx context.Context, k8sResourceType K8sVersionedResourceType, namespace string, k8sResourceName string, deleteK8sResourceFunction func(context.Context, string) error) error {
	logger.InfoC(ctx, "Deleting K8s %s with appVersion '%s' and name '%s' in namespace '%s'", k8sResourceType.Kind(), k8sResourceType.AppVersion(), k8sResourceName, namespace)

	err := deleteK8sResourceFunction(ctx, k8sResourceName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.WarnC(ctx, "Skip delete K8s %s with appVersion '%s' and name '%s' in namespace '%s' because it is not found", k8sResourceType.Kind(), k8sResourceType.AppVersion(), k8sResourceName, namespace)
			return nil
		} else {
			return LogError(logger, ctx, "Error deleting K8s %s with appVersion '%s' and name '%s' in namespace '%s': %v", k8sResourceType.Kind(), k8sResourceType.AppVersion(), k8sResourceName, namespace, err)
		}
	}

	logger.InfoC(ctx, "Successfully deleted K8s %s with appVersion '%s' and name '%s' in namespace '%s'", k8sResourceType.Kind(), k8sResourceType.AppVersion(), k8sResourceName, namespace)
	return nil
}
