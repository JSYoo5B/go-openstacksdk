package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DownloadImageRecord fetches complete owned metadata before transferring to
// memory, a borrowed writer, an SDK-owned file or an unread caller-owned stream.
func (c *Connection) DownloadImageRecord(ctx context.Context, input image.ImageRecordDownloadRequest, options ...image.ImageRecordDownloadOption) (*image.ImageRecordDownloadResult, error) {
	fail := func(err error) (*image.ImageRecordDownloadResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "DownloadImageRecord", Cause: err}
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
	return service.DownloadImageRecord(ctx, input, options...)
}
