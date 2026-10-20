package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GetImageMemberRecord fetches the selected member through the shared Glance
// service and returns its declared resource separately from actual response data.
func (c *Connection) GetImageMemberRecord(ctx context.Context, parent resource.Ref, input image.ImageMemberRecordRequest, options ...image.ImageMemberOption) (*image.ImageMemberRecord, error) {
	service, err := c.imageMemberRecordService(ctx, "GetImageMemberRecord")
	if err != nil {
		return nil, err
	}
	return service.GetImageMemberRecord(ctx, parent, input, options...)
}

// AddImageMemberRecord creates a member from concrete attributes through the
// shared Glance service. Sharing policy and required fields belong to the server.
func (c *Connection) AddImageMemberRecord(ctx context.Context, parent resource.Ref, options ...image.ImageMemberRecordWriteOption) (*image.ImageMemberRecord, error) {
	service, err := c.imageMemberRecordService(ctx, "AddImageMemberRecord")
	if err != nil {
		return nil, err
	}
	return service.AddImageMemberRecord(ctx, parent, options...)
}

// UpdateImageMemberRecord builds a fresh member request from the selected ID.
// The prior record supplies identity; its other attributes are not resubmitted.
func (c *Connection) UpdateImageMemberRecord(ctx context.Context, parent resource.Ref, input image.ImageMemberRecordRequest, options ...image.ImageMemberRecordWriteOption) (*image.ImageMemberRecord, error) {
	service, err := c.imageMemberRecordService(ctx, "UpdateImageMemberRecord")
	if err != nil {
		return nil, err
	}
	return service.UpdateImageMemberRecord(ctx, parent, input, options...)
}

// RemoveImageMemberRecord deletes the fixed parent/member target and retains
// actual accepted response evidence. A clean missing response is ignored by default.
func (c *Connection) RemoveImageMemberRecord(ctx context.Context, parent resource.Ref, input image.ImageMemberRecordRequest, options ...image.RemoveImageMemberOption) (*image.ImageMemberAcknowledgement, error) {
	service, err := c.imageMemberRecordService(ctx, "RemoveImageMemberRecord")
	if err != nil {
		return nil, err
	}
	return service.RemoveImageMemberRecord(ctx, parent, input, options...)
}

func (c *Connection) imageMemberRecordService(ctx context.Context, operation string) (*image.Service, error) {
	fail := func(err error) (*image.Service, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: operation, Cause: err}
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
	return service, nil
}
