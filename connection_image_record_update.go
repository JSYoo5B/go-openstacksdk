package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateImageRecord uses the Connection's shared Glance service, raw resource
// state and current cloud location. It performs no preliminary image lookup.
func (c *Connection) UpdateImageRecord(ctx context.Context, input image.ImageRecordUpdateRequest, options ...image.ImageRecordOption) (*image.ImageRecord, error) {
	fail := func(err error) (*image.ImageRecord, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "UpdateImageRecord", Cause: err}
	}
	if ctx == nil || c == nil {
		return fail(fmt.Errorf("%w: Connection and context are required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return fail(errors.Join(err, context.Cause(ctx)))
	}
	service, err := c.Image(ctx)
	if err != nil {
		return fail(err)
	}
	return service.UpdateImageRecord(ctx, input, options...)
}
