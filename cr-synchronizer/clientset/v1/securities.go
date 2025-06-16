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

type SecurityGetter interface {
	Security(namespace string) SecurityInterface
}

type SecurityInterface interface {
	Create(ctx context.Context, security *v1alpha1.Security, opts metav1.CreateOptions) (*v1alpha1.Security, error)
	Update(ctx context.Context, security *v1alpha1.Security, opts metav1.UpdateOptions) (*v1alpha1.Security, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Security, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.SecurityList, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.Security, err error)
	Apply(ctx context.Context, securityRevision *v1alpha1.SecurityApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.Security, err error)
}

type Securities struct {
	client rest.Interface
	ns     string
}

func newSecurityClient(c *V1Alpha1ClientClient, namespace string) *Securities {
	return &Securities{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

func (c *Securities) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1alpha1.Security, err error) {
	result = &v1alpha1.Security{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("securities").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *Securities) List(ctx context.Context, opts metav1.ListOptions) (result *v1alpha1.SecurityList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.SecurityList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("securities").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *Securities) Create(ctx context.Context, securityRevision *v1alpha1.Security, opts metav1.CreateOptions) (result *v1alpha1.Security, err error) {
	result = &v1alpha1.Security{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("securities").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(securityRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *Securities) Update(ctx context.Context, securityRevision *v1alpha1.Security, opts metav1.UpdateOptions) (result *v1alpha1.Security, err error) {
	result = &v1alpha1.Security{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("securities").
		Name(securityRevision.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(securityRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *Securities) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("securities").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *Securities) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("securities").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *Securities) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.Security, err error) {
	result = &v1alpha1.Security{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("securities").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

func (c *Securities) Apply(ctx context.Context, securityRevision *v1alpha1.SecurityApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.Security, err error) {
	if securityRevision == nil {
		return nil, fmt.Errorf("securityRevision provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(securityRevision)
	if err != nil {
		return nil, err
	}
	name := securityRevision.Name
	if name == nil {
		return nil, fmt.Errorf("securityRevision.Name must be provided to Apply")
	}
	result = &v1alpha1.Security{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("securities").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
