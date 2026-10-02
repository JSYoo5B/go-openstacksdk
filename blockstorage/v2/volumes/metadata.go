package volumes

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage/metadata"
	"gophercloudsdk/internal/cindermetadata"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// MetadataResult retains the actual metadata response and HTTP evidence.
type MetadataResult = metadata.Result

// MetadataDeleteResult retains each acknowledged key deletion or an actual clear response.
type MetadataDeleteResult = metadata.DeleteResult

// MetadataOption configures only library-owned request headers.
type MetadataOption = metadata.Option

func WithMetadataHeader(key, value string) MetadataOption { return metadata.WithHeader(key, value) }
func WithMetadataHeaders(headers map[string]string) MetadataOption {
	return metadata.WithHeaders(headers)
}

// MetadataScope fixes a parent ID and collection URL once. Response fields and
// later client URL changes cannot select a different metadata resource.
type MetadataScope struct{ inner *cindermetadata.Scope }

// MetadataIn uses explicit IDs without HTTP. An explicit Name is resolved once
// through the existing collection's exact lookup and duplicate-name policy.
func (a *API) MetadataIn(ctx context.Context, ref resource.Ref) (*MetadataScope, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	fail := func(err error) (*MetadataScope, error) {
		return nil, request.Wrap("MetadataIn", "volumes.metadata", err)
	}
	if err := cindermetadata.Validate(ctx, client); err != nil {
		return fail(err)
	}
	if err := ref.Validate(); err != nil {
		return fail(err)
	}
	id := ref.String()
	if ref.IsName() {
		if a.Resources == nil {
			return fail(fmt.Errorf("%w: parent collection is required for name resolution", resource.ErrInvalidOption))
		}
		var err error
		id, err = a.Resources.ResolveID(ctx, ref)
		if err != nil {
			return fail(err)
		}
	}
	if err := cindermetadata.Validate(ctx, client); err != nil {
		return fail(err)
	}
	inner, err := cindermetadata.New(ctx, client, "volumes", id)
	if err != nil {
		return fail(err)
	}
	return &MetadataScope{inner: inner}, nil
}

func (s *MetadataScope) runtime() *cindermetadata.Scope {
	if s == nil {
		return nil
	}
	return s.inner
}
func (s *MetadataScope) ID() string                            { return s.runtime().ID() }
func (s *MetadataScope) RawClient() *gophercloud.ServiceClient { return s.runtime().RawClient() }
func (s *MetadataScope) Get(ctx context.Context, options ...MetadataOption) (*MetadataResult, error) {
	return s.runtime().Get(ctx, options...)
}

// Merge sends an explicit metadata object, including for a nil or empty map.
func (s *MetadataScope) Merge(ctx context.Context, values map[string]string, options ...MetadataOption) (*MetadataResult, error) {
	return s.runtime().Merge(ctx, values, options...)
}

// Replace sends the complete replacement object; nil and empty maps clear it.
func (s *MetadataScope) Replace(ctx context.Context, values map[string]string, options ...MetadataOption) (*MetadataResult, error) {
	return s.runtime().Replace(ctx, values, options...)
}

// DeleteKeys clears with PUT for nil keys. A nonnil empty slice performs no
// HTTP; other keys are deleted in order, retaining prior acknowledgements on error.
func (s *MetadataScope) DeleteKeys(ctx context.Context, keys []string, options ...MetadataOption) (*MetadataDeleteResult, error) {
	return s.runtime().DeleteKeys(ctx, keys, options...)
}
