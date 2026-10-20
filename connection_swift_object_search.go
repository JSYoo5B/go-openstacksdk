package openstack

import (
	"context"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// SearchObjects shares the cached Swift service, collects its container listing
// and supplies the configured Connection location before cloud filtering.
// Explicit per-call location overrides remain owned; a reset DTO returns to the
// Connection default without asking users to build individual resource objects.
func (c *Connection) SearchObjects(ctx context.Context, container string, options ...objectstorage.ObjectSearchOption) (*objectstorage.ObjectSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchObjects", "objects", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ObjectStorageV1(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *objectstorage.ObjectSearchOpts) error {
		if cfg.Location == nil {
			value := location.Clone()
			cfg.Location = &value
		}
		return cloudread.Context(ctx)
	})
	result, err := service.Cloud.SearchObjects(ctx, container, owned...)
	return result, wrap(err)
}
