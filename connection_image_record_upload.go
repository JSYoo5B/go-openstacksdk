package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UploadImageRecord creates complete owned Glance metadata and uploads borrowed
// data through the shared service. The creation record and both actual phase
// receipts remain available; metadata is not fetched after upload.
func (c *Connection) UploadImageRecord(ctx context.Context, input image.ImageRecordUploadRequest, options ...image.ImageRecordUploadOption) (*image.ImageRecordUploadResult, error) {
	fail := func(err error) (*image.ImageRecordUploadResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "UploadImageRecord", Cause: err}
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
	return service.UploadImageRecord(ctx, input, options...)
}
