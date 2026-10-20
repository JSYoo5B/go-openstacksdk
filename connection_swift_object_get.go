package openstack

import (
	"context"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetObjectText supplies the cached Swift facade for eager owned text or output handling.
func (c *Connection) GetObjectText(ctx context.Context, container, object string, options ...objectstorage.ObjectGetOption) (*objectstorage.ObjectGetResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetObjectText", "objects", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ObjectStorageV1(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := service.Cloud.GetObjectText(ctx, container, object, slices.Clone(options)...)
	return value, wrap(err)
}
