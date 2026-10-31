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
