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

type MeshGetter interface {
	Mesh(namespace string) MeshInterface
}

type MeshInterface interface {
	Create(ctx context.Context, mesh *v1alpha1.Mesh, opts metav1.CreateOptions) (*v1alpha1.Mesh, error)
	Update(ctx context.Context, mesh *v1alpha1.Mesh, opts metav1.UpdateOptions) (*v1alpha1.Mesh, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Mesh, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.MeshList, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.Mesh, err error)
	Apply(ctx context.Context, meshRevision *v1alpha1.MeshApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.Mesh, err error)
}

type Meshes struct {
	client rest.Interface
	ns     string
}

func newMeshClient(c *V1Alpha1ClientClient, namespace string) *Meshes {
	return &Meshes{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

func (c *Meshes) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1alpha1.Mesh, err error) {
	log.Printf("delog: get mesh: %+v", name)
	result = &v1alpha1.Mesh{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("meshes").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *Meshes) List(ctx context.Context, opts metav1.ListOptions) (result *v1alpha1.MeshList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MeshList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("meshes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *Meshes) Create(ctx context.Context, meshRevision *v1alpha1.Mesh, opts metav1.CreateOptions) (result *v1alpha1.Mesh, err error) {
	log.Printf("delog: create mesh: %+v", meshRevision)
	result = &v1alpha1.Mesh{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("meshes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(meshRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *Meshes) Update(ctx context.Context, meshRevision *v1alpha1.Mesh, opts metav1.UpdateOptions) (result *v1alpha1.Mesh, err error) {
	result = &v1alpha1.Mesh{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("meshes").
		Name(meshRevision.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(meshRevision).
		Do(ctx).
		Into(result)
	return
}

func (c *Meshes) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("meshes").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *Meshes) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("meshes").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *Meshes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.Mesh, err error) {
	result = &v1alpha1.Mesh{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("meshes").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

func (c *Meshes) Apply(ctx context.Context, meshRevision *v1alpha1.MeshApplyConfiguration, opts metav1.ApplyOptions) (result *v1alpha1.Mesh, err error) {
	log.Printf("delog: apply mesh: %+v", meshRevision)
	if meshRevision == nil {
		return nil, fmt.Errorf("meshRevision provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(meshRevision)
	if err != nil {
		return nil, err
	}
	name := meshRevision.Name
	if name == nil {
		return nil, fmt.Errorf("meshRevision.Name must be provided to Apply")
	}
	result = &v1alpha1.Mesh{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("meshes").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
