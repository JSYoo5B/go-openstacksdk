package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImportImageRecord submits one owned import through the shared Glance service.
// The SDK-produced record retains private formats and pending state, and the
// actual asynchronous acknowledgement is separate from that prepared record.
func (c *Connection) ImportImageRecord(ctx context.Context, input image.ImageRecordImportRequest, options ...image.ImageRecordImportOption) (*image.ImageRecordImportResult, error) {
	fail := func(err error) (*image.ImageRecordImportResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "ImportImageRecord", Cause: err}
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
	return service.ImportImageRecord(ctx, input, options...)
}
