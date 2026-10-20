package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateImageRecord uses the connection's merged image policy and resolves
// Swift only when the configured Task branch requires it.
func (c *Connection) CreateImageRecord(ctx context.Context, input image.ImageRecordCreateRequest, options ...image.ImageRecordCreateOption) (*image.ImageRecordCreateResult, error) {
	fail := func(err error) (*image.ImageRecordCreateResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "CreateImageRecord", Cause: err}
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
	return service.CreateImageRecord(ctx, input, options...)
}
