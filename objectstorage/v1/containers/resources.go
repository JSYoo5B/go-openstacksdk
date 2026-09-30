package containers

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"gophercloudsdk/resource"
)

// ContainerResource joins a Swift list record with metadata from HEAD. Details,
// Metadata and Header are populated by Get; list records contain counts and name.
type ContainerResource struct {
	Container
	Details  *GetHeader
	Metadata map[string]string
	Header   http.Header
}

func validateContainerID(name string) error {
	if err := swift.CheckContainerName(name); err != nil {
		return fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	return nil
}

func (a *API) newResources() *resource.Collection[ContainerResource] {
	return resource.NewCollection(resource.Adapter[ContainerResource]{
		Kind: "container", ValidateID: validateContainerID,
		Get: func(ctx context.Context, name string) (*ContainerResource, error) {
			result := upstream.Get(ctx, a.client, name, nil)
			header, err := result.Extract()
			if err != nil {
				return nil, err
			}
			metadata, err := result.ExtractMetadata()
			if err != nil {
				return nil, err
			}
			return &ContainerResource{
				Container: Container{Name: name, Count: header.ObjectCount, Bytes: header.BytesUsed},
				Details:   header, Metadata: metadata, Header: result.Header.Clone(),
			}, nil
		},
		Iterate: func(ctx context.Context, query url.Values) iter.Seq2[*ContainerResource, error] {
			return func(yield func(*ContainerResource, error) bool) {
				var options []ListOption
				for key, values := range query {
					for _, value := range values {
						options = append(options, WithListQuery(key, value))
					}
				}
				for item, err := range a.List(ctx, options...) {
					if err != nil {
						yield(nil, err)
						return
					}
					if !yield(&ContainerResource{Container: *item}, nil) {
						return
					}
				}
			}
		},
		Delete: func(ctx context.Context, name string) error { _, err := a.Delete(ctx, name); return err },
		ID:     func(item *ContainerResource) string { return item.Name }, Name: func(item *ContainerResource) string { return item.Name },
		NameQueryKey: "prefix", NameQuery: func(name string) string { return name },
	})
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*ContainerResource, error) {
	return a.Resources.Find(ctx, ref, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*ContainerResource, error) {
	return a.Resources.All(ctx, options...)
}
