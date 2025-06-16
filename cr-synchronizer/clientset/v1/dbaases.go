package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/netcracker/cr-synchronizer/api/types/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type DbaasGetter interface {
	DBaaS(namespace string) DbaasInterface
}

type DbaasInterface interface {
	Create(ctx context.Context, dbaas *v1alpha1.DBaaS, opts metav1.CreateOptions) (*v1alpha1.DBaaS, error)
	Update(ctx context.Context, dbaas *v1alpha1.DBaaS, opts metav1.UpdateOptions) (*v1alpha1.DBaaS, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.DBaaS, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.DBaaSList, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.DBaaS, err error)
	Apply(ctx context.Context, dbaasRevision *v1alpha1.DBaaSApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.DBaaS, err error)
}

type DBaaSes struct {
	client rest.Interface
	ns     string
}

func newDbaasClient(c *V1Alpha1ClientClient, namespace string) *DBaaSes {
	return &DBaaSes{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

func (c *DBaaSes) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1alpha1.DBaaS, err error) {
	result = &v1alpha1.DBaaS{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("dbaases").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *DBaaSes) List(ctx context.Context, opts metav1.ListOptions) (result *v1alpha1.DBaaSList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.DBaaSList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("dbaases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *DBaaSes) Create(ctx context.Context, dbaasRevision *v1alpha1.DBaaS, opts metav1.CreateOptions) (result *v1alpha1.DBaaS, err error) {
	result = &v1alpha1.DBaaS{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("dbaases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(dbaasRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *DBaaSes) Update(ctx context.Context, dbaasRevision *v1alpha1.DBaaS, opts metav1.UpdateOptions) (result *v1alpha1.DBaaS, err error) {
	result = &v1alpha1.DBaaS{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("dbaases").
		Name(dbaasRevision.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(dbaasRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *DBaaSes) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("dbaases").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *DBaaSes) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("dbaases").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *DBaaSes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.DBaaS, err error) {
	result = &v1alpha1.DBaaS{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("dbaases").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

func (c *DBaaSes) Apply(ctx context.Context, dbaasRevision *v1alpha1.DBaaSApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.DBaaS, err error) {
	if dbaasRevision == nil {
		return nil, fmt.Errorf("dbaasRevision provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(dbaasRevision)
	if err != nil {
		return nil, err
	}
	name := dbaasRevision.Name
	if name == nil {
		return nil, fmt.Errorf("dbaasRevision.Name must be provided to Apply")
	}
	result = &v1alpha1.DBaaS{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("dbaases").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
