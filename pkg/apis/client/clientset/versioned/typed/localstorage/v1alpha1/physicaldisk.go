// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	scheme "github.com/hwameiStor/local-storage/pkg/apis/client/clientset/versioned/scheme"
	v1alpha1 "github.com/hwameiStor/local-storage/pkg/apis/localstorage/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PhysicalDisksGetter has a method to return a PhysicalDiskInterface.
// A group's client should implement this interface.
type PhysicalDisksGetter interface {
	PhysicalDisks() PhysicalDiskInterface
}

// PhysicalDiskInterface has methods to work with PhysicalDisk resources.
type PhysicalDiskInterface interface {
	Create(ctx context.Context, physicalDisk *v1alpha1.PhysicalDisk, opts v1.CreateOptions) (*v1alpha1.PhysicalDisk, error)
	Update(ctx context.Context, physicalDisk *v1alpha1.PhysicalDisk, opts v1.UpdateOptions) (*v1alpha1.PhysicalDisk, error)
	UpdateStatus(ctx context.Context, physicalDisk *v1alpha1.PhysicalDisk, opts v1.UpdateOptions) (*v1alpha1.PhysicalDisk, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.PhysicalDisk, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.PhysicalDiskList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.PhysicalDisk, err error)
	PhysicalDiskExpansion
}

// physicalDisks implements PhysicalDiskInterface
type physicalDisks struct {
	client rest.Interface
}

// newPhysicalDisks returns a PhysicalDisks
func newPhysicalDisks(c *LocalStorageV1alpha1Client) *physicalDisks {
	return &physicalDisks{
		client: c.RESTClient(),
	}
}

// Get takes name of the physicalDisk, and returns the corresponding physicalDisk object, and an error if there is any.
func (c *physicalDisks) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.PhysicalDisk, err error) {
	result = &v1alpha1.PhysicalDisk{}
	err = c.client.Get().
		Resource("physicaldisks").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PhysicalDisks that match those selectors.
func (c *physicalDisks) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.PhysicalDiskList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.PhysicalDiskList{}
	err = c.client.Get().
		Resource("physicaldisks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested physicalDisks.
func (c *physicalDisks) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("physicaldisks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a physicalDisk and creates it.  Returns the server's representation of the physicalDisk, and an error, if there is any.
func (c *physicalDisks) Create(ctx context.Context, physicalDisk *v1alpha1.PhysicalDisk, opts v1.CreateOptions) (result *v1alpha1.PhysicalDisk, err error) {
	result = &v1alpha1.PhysicalDisk{}
	err = c.client.Post().
		Resource("physicaldisks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(physicalDisk).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a physicalDisk and updates it. Returns the server's representation of the physicalDisk, and an error, if there is any.
func (c *physicalDisks) Update(ctx context.Context, physicalDisk *v1alpha1.PhysicalDisk, opts v1.UpdateOptions) (result *v1alpha1.PhysicalDisk, err error) {
	result = &v1alpha1.PhysicalDisk{}
	err = c.client.Put().
		Resource("physicaldisks").
		Name(physicalDisk.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(physicalDisk).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *physicalDisks) UpdateStatus(ctx context.Context, physicalDisk *v1alpha1.PhysicalDisk, opts v1.UpdateOptions) (result *v1alpha1.PhysicalDisk, err error) {
	result = &v1alpha1.PhysicalDisk{}
	err = c.client.Put().
		Resource("physicaldisks").
		Name(physicalDisk.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(physicalDisk).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the physicalDisk and deletes it. Returns an error if one occurs.
func (c *physicalDisks) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("physicaldisks").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *physicalDisks) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("physicaldisks").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched physicalDisk.
func (c *physicalDisks) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.PhysicalDisk, err error) {
	result = &v1alpha1.PhysicalDisk{}
	err = c.client.Patch(pt).
		Resource("physicaldisks").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
