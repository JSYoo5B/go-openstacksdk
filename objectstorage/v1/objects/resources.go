package objects

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// ObjectResource joins a Swift object listing with metadata from HEAD. Container
// is always present; Details, Metadata and Header are populated by Get.
type ObjectResource struct {
	Object
	Container string
	Details   *GetHeader
	Metadata  map[string]string
	Header    http.Header
}

type ObjectScope struct {
	*resource.Collection[ObjectResource]
	api       *API
	container string
}

func validateObjectID(name string) error {
	if err := swift.CheckObjectName(name); err != nil {
		return fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	return nil
}

// InContainer resolves the parent once. Explicit identifiers bypass lookup;
// both container names and object keys are escaped by the upstream URL helpers.
func (a *API) InContainer(ctx context.Context, ref resource.Ref) (*ObjectScope, error) {
	parent, err := containers.New(a.client).Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, err
	}
	scope := &ObjectScope{api: a, container: parent}
	scope.Collection = resource.NewCollection(resource.Adapter[ObjectResource]{
		Kind: "object", ValidateID: validateObjectID,
		Get: func(ctx context.Context, name string) (*ObjectResource, error) {
			result := upstream.Get(ctx, a.client, parent, name, nil)
			header, err := result.Extract()
			if err != nil {
				return nil, err
			}
			metadata, err := result.ExtractMetadata()
			if err != nil {
				return nil, err
			}
			return &ObjectResource{
				Object:    Object{Name: name, Bytes: header.ContentLength, ContentType: header.ContentType, Hash: header.ETag, LastModified: header.LastModified, VersionID: header.ObjectVersionID},
				Container: parent, Details: header, Metadata: metadata, Header: result.Header.Clone(),
			}, nil
		},
		Iterate: func(ctx context.Context, query url.Values) iter.Seq2[*ObjectResource, error] {
			return scope.listResources(ctx, query, resource.ListControl{})
		},
		IterateControlled: scope.listResources,
		Delete:            func(ctx context.Context, name string) error { _, err := a.Delete(ctx, parent, name); return err },
		ID:                func(item *ObjectResource) string { return item.Name }, Name: func(item *ObjectResource) string { return item.Name },
		NameQueryKey: "prefix", NameQuery: func(name string) string { return name },
	})
	return scope, nil
}

func (s *ObjectScope) listResources(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*ObjectResource, error] {
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
		return func(yield func(*ObjectResource, error) bool) { yield(nil, request.Wrap("List", "objects", err)) }
	}
	builder := listOptsBuilder{base: config.Options, config: config}
	return resource.StreamWithControl(ctx, upstream.List(s.api.client, s.container, builder), func(page pagination.Page) ([]ObjectResource, error) {
		items, err := upstream.ExtractInfo(page)
		if err != nil {
			return nil, err
		}
		values := make([]ObjectResource, len(items))
		for i, item := range items {
			values[i] = ObjectResource{Object: item, Container: s.container}
		}
		return values, nil
	}, control)
}

func (s *ObjectScope) Create(ctx context.Context, name string, opts CreateOpts, options ...CreateOption) (*CreateHeader, error) {
	return s.api.Create(ctx, s.container, name, opts, options...)
}

func (s *ObjectScope) Update(ctx context.Context, ref resource.Ref, opts UpdateOpts, options ...UpdateOption) (*UpdateHeader, error) {
	name, err := s.ResolveID(ctx, ref)
	if err != nil {
		return nil, err
	}
	return s.api.Update(ctx, s.container, name, opts, options...)
}

func (s *ObjectScope) Download(ctx context.Context, ref resource.Ref, options ...DownloadOption) (*request.Download[*DownloadHeader], error) {
	name, err := s.ResolveID(ctx, ref)
	if err != nil {
		return nil, err
	}
	return s.api.Download(ctx, s.container, name, options...)
}

func (s *ObjectScope) Copy(ctx context.Context, ref resource.Ref, opts CopyOpts, options ...CopyOption) (*CopyHeader, error) {
	name, err := s.ResolveID(ctx, ref)
	if err != nil {
		return nil, err
	}
	return s.api.Copy(ctx, s.container, name, opts, options...)
}
