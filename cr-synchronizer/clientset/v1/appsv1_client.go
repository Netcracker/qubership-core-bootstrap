package v1

import (
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
