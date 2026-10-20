package openstack

import (
	"context"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// SearchContainers shares the cached Swift service and supplies configured
// Connection location before identifier/mapping/expression selection. Explicit
// per-call typed location overrides remain owned; a reset DTO uses the Connection
// default without requiring callers to construct a resource or builder.
func (c *Connection) SearchContainers(ctx context.Context, options ...objectstorage.ContainerSearchOption) (*objectstorage.ContainerSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchContainers", "containers", cloudread.ContextError(ctx, err))
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
	owned := append(slices.Clone(options), func(cfg *objectstorage.ContainerSearchOpts) error {
		if cfg.Location == nil {
			value := location.Clone()
			cfg.Location = &value
		}
		return cloudread.Context(ctx)
	})
	result, err := service.Cloud.SearchContainers(ctx, owned...)
	return result, wrap(err)
}
