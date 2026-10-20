package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// StageImageRecord stages a queued SDK-produced image record and performs its
// required metadata fetch through the shared Glance service. Binary data is
// borrowed at its current position, or opened from the SDK-owned filename.
func (c *Connection) StageImageRecord(ctx context.Context, input image.ImageRecordStageRequest, options ...image.ImageRecordStageOption) (*image.ImageRecordStageResult, error) {
	fail := func(err error) (*image.ImageRecordStageResult, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: "StageImageRecord", Cause: err}
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
	return service.StageImageRecord(ctx, input, options...)
}
