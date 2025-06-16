package clientset

import (
	"fmt"
	"net/http"

	ncv1 "github.com/netcracker/cr-synchronizer/clientset/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	NetcrackerV1() ncv1.V1Alpha1ClientInterface
	CoreV1() corev1.CoreV1Interface
	AppsV1() appsv1.AppsV1Interface
	BatchV1() batchv1.BatchV1Interface
}

type Clientset struct {
	ncV1    *ncv1.V1Alpha1ClientClient
	coreV1  *corev1.CoreV1Client
	appsV1  *appsv1.AppsV1Client
	batchV1 *batchv1.BatchV1Client
}

func (c *Clientset) NetcrackerV1() ncv1.V1Alpha1ClientInterface {
	return c.ncV1
}

func (c *Clientset) CoreV1() corev1.CoreV1Interface {
	return c.coreV1
}

func (c *Clientset) AppsV1() appsv1.AppsV1Interface {
	return c.appsV1
}

func (c *Clientset) BatchV1() batchv1.BatchV1Interface {
	return c.batchV1
}

func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	httpClient, err := rest.HTTPClientFor(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&configShallowCopy, httpClient)
}

func NewForConfigAndClient(c *rest.Config, httpClient *http.Client) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		if configShallowCopy.Burst <= 0 {
			return nil, fmt.Errorf("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.coreV1, err = corev1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.ncV1, err = ncv1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.batchV1, err = batchv1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.appsV1, err = appsv1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}
