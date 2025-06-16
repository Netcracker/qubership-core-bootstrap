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

type CompositeGetter interface {
	Composite(namespace string) CompositeInterface
}

type CompositeInterface interface {
	Create(ctx context.Context, composite *v1alpha1.Composite, opts metav1.CreateOptions) (*v1alpha1.Composite, error)
	Update(ctx context.Context, composite *v1alpha1.Composite, opts metav1.UpdateOptions) (*v1alpha1.Composite, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Composite, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.CompositeList, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.Composite, err error)
	Apply(ctx context.Context, compositeRevision *v1alpha1.CompositeApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.Composite, err error)
}

type Composites struct {
	client rest.Interface
	ns     string
}

func newCompositeClient(c *V1Alpha1ClientClient, namespace string) *Composites {
	return &Composites{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

func (c *Composites) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1alpha1.Composite, err error) {
	result = &v1alpha1.Composite{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("composites").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *Composites) List(ctx context.Context, opts metav1.ListOptions) (result *v1alpha1.CompositeList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.CompositeList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("composites").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *Composites) Create(ctx context.Context, compositeRevision *v1alpha1.Composite, opts metav1.CreateOptions) (result *v1alpha1.Composite, err error) {
	result = &v1alpha1.Composite{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("composites").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(compositeRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *Composites) Update(ctx context.Context, compositeRevision *v1alpha1.Composite, opts metav1.UpdateOptions) (result *v1alpha1.Composite, err error) {
	result = &v1alpha1.Composite{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("composites").
		Name(compositeRevision.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(compositeRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *Composites) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("composites").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *Composites) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("composites").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *Composites) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.Composite, err error) {
	result = &v1alpha1.Composite{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("composites").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

func (c *Composites) Apply(ctx context.Context, compositeRevision *v1alpha1.CompositeApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.Composite, err error) {
	if compositeRevision == nil {
		return nil, fmt.Errorf("compositeRevision provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(compositeRevision)
	if err != nil {
		return nil, err
	}
	name := compositeRevision.Name
	if name == nil {
		return nil, fmt.Errorf("compositeRevision.Name must be provided to Apply")
	}
	result = &v1alpha1.Composite{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("composites").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
