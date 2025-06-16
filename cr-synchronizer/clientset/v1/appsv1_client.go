package v1

import (
	"net/http"

	v1alpha1 "github.com/netcracker/cr-synchronizer/api/types/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type V1Alpha1ClientInterface interface {
	RESTClient() rest.Interface
	MaasGetter
	DbaasGetter
	MeshGetter
	CompositeGetter
	SecurityGetter
}

type V1Alpha1ClientClient struct {
	restClient rest.Interface
}

func setConfigDefaults(config *rest.Config) error {
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: v1alpha1.GroupName, Version: v1alpha1.GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	return nil
}

func NewForConfig(c *rest.Config) (*V1Alpha1ClientClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

func NewForConfigAndClient(c *rest.Config, h *http.Client) (*V1Alpha1ClientClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &V1Alpha1ClientClient{restClient: client}, nil
}

func (c *V1Alpha1ClientClient) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}

func (c *V1Alpha1ClientClient) MaaS(namespace string) MaasInterface {
	return newMaasClient(c, namespace)
}

func (c *V1Alpha1ClientClient) DBaaS(namespace string) DbaasInterface {
	return newDbaasClient(c, namespace)
}

func (c *V1Alpha1ClientClient) Mesh(namespace string) MeshInterface {
	return newMeshClient(c, namespace)
}

func (c *V1Alpha1ClientClient) Composite(namespace string) CompositeInterface {
	return newCompositeClient(c, namespace)
}

func (c *V1Alpha1ClientClient) Security(namespace string) SecurityInterface {
	return newSecurityClient(c, namespace)
}
