package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DeleteImageRecord deletes a whole image or one store copy through the shared
// Glance service, keeping the local record separate from actual response data.
func (c *Connection) DeleteImageRecord(ctx context.Context, input image.ImageRecordDeleteRequest, options ...image.ImageRecordDeleteOption) (*image.ImageRecordDeleteResult, error) {
	fail := func(err error) (*image.ImageRecordDeleteResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "DeleteImageRecord", Cause: err}
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
	return service.DeleteImageRecord(ctx, input, options...)
}
