package containers

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/pagination"
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
			return a.listResources(ctx, query, resource.ListControl{})
		},
		IterateControlled: a.listResources,
		Delete:            func(ctx context.Context, name string) error { _, err := a.Delete(ctx, name); return err },
		ID:                func(item *ContainerResource) string { return item.Name }, Name: func(item *ContainerResource) string { return item.Name },
		NameQueryKey: "prefix", NameQuery: func(name string) string { return name },
	})
}

func (a *API) listResources(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*ContainerResource, error] {
	options := make([]ListOption, 0, len(query))
	for key, values := range query {
		for _, value := range values {
			options = append(options, WithListQuery(key, value))
		}
	}
	config, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, false)
	}
	if err != nil {
		return func(yield func(*ContainerResource, error) bool) { yield(nil, request.Wrap("List", "containers", err)) }
	}
	builder := listOptsBuilder{base: config.Options, config: config}
	return resource.StreamWithControl(ctx, upstream.List(a.client, builder), func(page pagination.Page) ([]ContainerResource, error) {
		items, err := upstream.ExtractInfo(page)
		if err != nil {
			return nil, err
		}
		values := make([]ContainerResource, len(items))
		for i, item := range items {
			values[i] = ContainerResource{Container: item}
		}
		return values, nil
	}, control)
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*ContainerResource, error) {
	return a.Resources.Find(ctx, ref, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*ContainerResource, error) {
	return a.Resources.All(ctx, options...)
}
