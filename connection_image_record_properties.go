package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateImagePropertiesRecord applies the owned Glance property helper through
// the shared service, including its kernel/ramdisk lookup and conversion policy.
// Updated follows the helper's local decision and does not imply a PATCH was sent.
func (c *Connection) UpdateImagePropertiesRecord(ctx context.Context, input image.ImageRecordPropertiesRequest, options ...image.ImageRecordPropertiesOption) (*image.ImageRecordPropertiesResult, error) {
	fail := func(err error) (*image.ImageRecordPropertiesResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "UpdateImagePropertiesRecord", Cause: err}
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
	return service.UpdateImagePropertiesRecord(ctx, input, options...)
}
