package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"time"

	v1alpha1 "github.com/netcracker/cr-synchronizer/api/types/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type MaasGetter interface {
	MaaS(namespace string) MaasInterface
}

type MaasInterface interface {
	Create(ctx context.Context, maas *v1alpha1.MaaS, opts metav1.CreateOptions) (*v1alpha1.MaaS, error)
	Update(ctx context.Context, maas *v1alpha1.MaaS, opts metav1.UpdateOptions) (*v1alpha1.MaaS, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.MaaS, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.MaaSList, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.MaaS, err error)
	Apply(ctx context.Context, maasRevision *v1alpha1.MaaSApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.MaaS, err error)
}

type MaaSes struct {
	client rest.Interface
	ns     string
}

func newMaasClient(c *V1Alpha1ClientClient, namespace string) *MaaSes {
	return &MaaSes{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

func (c *MaaSes) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1alpha1.MaaS, err error) {
	log.Printf("delog: get maas: %+v", name)
	result = &v1alpha1.MaaS{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("maases").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *MaaSes) List(ctx context.Context, opts metav1.ListOptions) (result *v1alpha1.MaaSList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MaaSList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("maases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *MaaSes) Create(ctx context.Context, maasRevision *v1alpha1.MaaS, opts metav1.CreateOptions) (result *v1alpha1.MaaS, err error) {
	log.Printf("delog: create maas: %+v", maasRevision)

	result = &v1alpha1.MaaS{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("maases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(maasRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *MaaSes) Update(ctx context.Context, maasRevision *v1alpha1.MaaS, opts metav1.UpdateOptions) (result *v1alpha1.MaaS, err error) {
	result = &v1alpha1.MaaS{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("maases").
		Name(maasRevision.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(maasRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *MaaSes) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("maases").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *MaaSes) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("maases").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *MaaSes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.MaaS, err error) {
	result = &v1alpha1.MaaS{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("maases").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

func (c *MaaSes) Apply(ctx context.Context, maasRevision *v1alpha1.MaaSApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.MaaS, err error) {
	log.Printf("delog: apply maas: %+v", maasRevision)

	if maasRevision == nil {
		return nil, fmt.Errorf("MaaS provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(maasRevision)
	if err != nil {
		return nil, err
	}
	name := maasRevision.Name
	if name == nil {
		return nil, fmt.Errorf("MaaS.Name must be provided to Apply")
	}
	result = &v1alpha1.MaaS{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("maases").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
