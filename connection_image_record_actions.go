package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DeactivateImageRecord submits a fixed image action through the shared Glance
// service and returns the prepared local record separately from its receipt.
func (c *Connection) DeactivateImageRecord(ctx context.Context, input image.ImageRecordActionRequest, options ...image.ImageRecordActionOption) (*image.ImageRecordActionResult, error) {
	fail := func(err error) (*image.ImageRecordActionResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "DeactivateImageRecord", Cause: err}
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
	return service.DeactivateImageRecord(ctx, input, options...)
}

// ReactivateImageRecord submits the complementary action with the same owned
// inputs and response policy. It does not poll or infer a new image status.
func (c *Connection) ReactivateImageRecord(ctx context.Context, input image.ImageRecordActionRequest, options ...image.ImageRecordActionOption) (*image.ImageRecordActionResult, error) {
	fail := func(err error) (*image.ImageRecordActionResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "ReactivateImageRecord", Cause: err}
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
	return service.ReactivateImageRecord(ctx, input, options...)
}
