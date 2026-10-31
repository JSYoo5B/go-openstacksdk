package openstack

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudimage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"
	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// WithImageCreatePolicy overrides merged cloud image settings with concrete SDK
// options. Defaults, service construction and extension handling remain owned.
func WithImageCreatePolicy(options ...image.ImageCreatePolicyOption) ConnectionOption {
	captured := append([]image.ImageCreatePolicyOption(nil), options...)
	return func(config *connectionOptions) error {
		policy, err := image.PrepareImageCreatePolicy(captured...)
		if err == nil {
			config.imageCreatePolicy, config.imageCreatePolicySet = policy, true
		}
		return err
	}
}

func (c *Connection) cloudImageDependencies() cloudimage.Dependencies {
	if c == nil {
		return cloudimage.Dependencies{}
	}
	return cloudimage.Dependencies{
		Image:         c.Image,
		ObjectStorage: c.ObjectStorageV1,
		Provider:      func() *gophercloud.ProviderClient { return c.provider },
	}
}

// Only the configured Task branch resolves Swift. Source-only registration lets
// the Glance operation observe later Swift binding changes without recursion.
func (c *Connection) imageCreateSwiftService(ctx context.Context) (*objectapi.Service, error) {
	return cloudimage.SwiftService(ctx, c.cloudImageDependencies())
}

// CloudImageCreateRequest mirrors Cloud create_image. An empty VolumeID uses
// the Glance CreateImageRecord workflow with Image; otherwise the volume branch
// uses Image.Name with the Cinder Proxy create_image. PollInterval is a Go
// extension for the Cloud wait loop and its timeout cleanup; zero means the
// fixed two-second iterate_timeout interval.
type CloudImageCreateRequest struct {
	Image        image.ImageRecordCreateRequest
	VolumeID     string
	PollInterval time.Duration
}

// CloudImageCreateResult keeps every Cloud create_image phase. Image is the
// Source return value: the created record without wait, or the first found
// record whose status is neither queued nor saving.
type CloudImageCreateResult struct {
	Image       *image.ImageRecord
	Created     *image.ImageRecordCreateResult
	Volume      *VolumeImageCreateResult
	WaitLookups int
	WaitLast    *image.ImageRecord
	Cleanup     *image.ImageRecordCloudDeleteResult
}

// CreateCloudImageRecord implements Cloud create_image. Without a volume it
// runs CreateImageRecord with the same options; with a volume it runs the
// Cinder Proxy create_image using the allow_duplicates and format options.
// When wait is truthy it then repeats Cloud get_image on the created id until
// a found status is neither queued nor saving. On timeout it runs Cloud
// delete_image(id, wait=True) and returns the timeout with that cleanup.
func (c *Connection) CreateCloudImageRecord(ctx context.Context, input CloudImageCreateRequest, options ...image.ImageRecordCreateOption) (*CloudImageCreateResult, error) {
	createVolume := func(ctx context.Context, input cloudvolume.ImageCreateRequest) (*VolumeImageCreateResult, *image.ImageRecord, error) {
		result, err := c.CreateVolumeImageRecord(ctx, VolumeImageCreateRequest(input))
		if result == nil {
			return nil, nil, err
		}
		return result, result.Image, err
	}
	result, err := cloudimage.Create(ctx, c.cloudImageDependencies(), createVolume, cloudimage.CreateRequest(input), options...)
	return (*CloudImageCreateResult)(result), err
}

// CreateImageRecord uses the connection's merged image policy and resolves
// Swift only when the configured Task branch requires it.
func (c *Connection) CreateImageRecord(ctx context.Context, input image.ImageRecordCreateRequest, options ...image.ImageRecordCreateOption) (*image.ImageRecordCreateResult, error) {
	service, err := c.imageRecordService(ctx, "CreateImageRecord")
	if err != nil {
		return nil, err
	}
	return service.CreateImageRecord(ctx, input, options...)
}

// UpdateImageRecord uses the Connection's shared Glance service, raw resource
// state and current cloud location. It performs no preliminary image lookup.
func (c *Connection) UpdateImageRecord(ctx context.Context, input image.ImageRecordUpdateRequest, options ...image.ImageRecordOption) (*image.ImageRecord, error) {
	service, err := c.imageRecordService(ctx, "UpdateImageRecord")
	if err != nil {
		return nil, err
	}
	return service.UpdateImageRecord(ctx, input, options...)
}

// UpdateImagePropertiesRecord applies the owned Glance property helper through
// the shared service, including its kernel/ramdisk lookup and conversion policy.
// Updated follows the helper's local decision and does not imply a PATCH was sent.
func (c *Connection) UpdateImagePropertiesRecord(ctx context.Context, input image.ImageRecordPropertiesRequest, options ...image.ImageRecordPropertiesOption) (*image.ImageRecordPropertiesResult, error) {
	service, err := c.imageRecordService(ctx, "UpdateImagePropertiesRecord")
	if err != nil {
		return nil, err
	}
	return service.UpdateImagePropertiesRecord(ctx, input, options...)
}

// DeleteImageRecord deletes a whole image or one store copy through the shared
// Glance service, keeping the local record separate from actual response data.
func (c *Connection) DeleteImageRecord(ctx context.Context, input image.ImageRecordDeleteRequest, options ...image.ImageRecordDeleteOption) (*image.ImageRecordDeleteResult, error) {
	service, err := c.imageRecordService(ctx, "DeleteImageRecord")
	if err != nil {
		return nil, err
	}
	return service.DeleteImageRecord(ctx, input, options...)
}

// UploadImageRecord creates complete owned Glance metadata and uploads borrowed
// data through the shared service. The creation record and both actual phase
// receipts remain available; metadata is not fetched after upload.
func (c *Connection) UploadImageRecord(ctx context.Context, input image.ImageRecordUploadRequest, options ...image.ImageRecordUploadOption) (*image.ImageRecordUploadResult, error) {
	service, err := c.imageRecordService(ctx, "UploadImageRecord")
	if err != nil {
		return nil, err
	}
	return service.UploadImageRecord(ctx, input, options...)
}

// StageImageRecord stages a queued SDK-produced image record and performs its
// required metadata fetch through the shared Glance service. Binary data is
// borrowed at its current position, or opened from the SDK-owned filename.
func (c *Connection) StageImageRecord(ctx context.Context, input image.ImageRecordStageRequest, options ...image.ImageRecordStageOption) (*image.ImageRecordStageResult, error) {
	service, err := c.imageRecordService(ctx, "StageImageRecord")
	if err != nil {
		return nil, err
	}
	return service.StageImageRecord(ctx, input, options...)
}

// ImportImageRecord submits one owned import through the shared Glance service.
// The SDK-produced record retains private formats and pending state, and the
// actual asynchronous acknowledgement is separate from that prepared record.
func (c *Connection) ImportImageRecord(ctx context.Context, input image.ImageRecordImportRequest, options ...image.ImageRecordImportOption) (*image.ImageRecordImportResult, error) {
	service, err := c.imageRecordService(ctx, "ImportImageRecord")
	if err != nil {
		return nil, err
	}
	return service.ImportImageRecord(ctx, input, options...)
}

// DownloadImageRecord fetches complete owned metadata before transferring to
// memory, a borrowed writer, an SDK-owned file or an unread caller-owned stream.
func (c *Connection) DownloadImageRecord(ctx context.Context, input image.ImageRecordDownloadRequest, options ...image.ImageRecordDownloadOption) (*image.ImageRecordDownloadResult, error) {
	service, err := c.imageRecordService(ctx, "DownloadImageRecord")
	if err != nil {
		return nil, err
	}
	return service.DownloadImageRecord(ctx, input, options...)
}

// DeactivateImageRecord submits a fixed image action through the shared Glance
// service and returns the prepared local record separately from its receipt.
func (c *Connection) DeactivateImageRecord(ctx context.Context, input image.ImageRecordActionRequest, options ...image.ImageRecordActionOption) (*image.ImageRecordActionResult, error) {
	service, err := c.imageRecordService(ctx, "DeactivateImageRecord")
	if err != nil {
		return nil, err
	}
	return service.DeactivateImageRecord(ctx, input, options...)
}

// ReactivateImageRecord submits the complementary action with the same owned
// inputs and response policy. It does not poll or infer a new image status.
func (c *Connection) ReactivateImageRecord(ctx context.Context, input image.ImageRecordActionRequest, options ...image.ImageRecordActionOption) (*image.ImageRecordActionResult, error) {
	service, err := c.imageRecordService(ctx, "ReactivateImageRecord")
	if err != nil {
		return nil, err
	}
	return service.ReactivateImageRecord(ctx, input, options...)
}

// GetImageMemberRecord fetches the selected member through the shared Glance
// service and returns its declared resource separately from actual response data.
func (c *Connection) GetImageMemberRecord(ctx context.Context, parent resource.Ref, input image.ImageMemberRecordRequest, options ...image.ImageMemberOption) (*image.ImageMemberRecord, error) {
	service, err := c.imageRecordService(ctx, "GetImageMemberRecord")
	if err != nil {
		return nil, err
	}
	return service.GetImageMemberRecord(ctx, parent, input, options...)
}

// AddImageMemberRecord creates a member from concrete attributes through the
// shared Glance service. Sharing policy and required fields belong to the server.
func (c *Connection) AddImageMemberRecord(ctx context.Context, parent resource.Ref, options ...image.ImageMemberRecordWriteOption) (*image.ImageMemberRecord, error) {
	service, err := c.imageRecordService(ctx, "AddImageMemberRecord")
	if err != nil {
		return nil, err
	}
	return service.AddImageMemberRecord(ctx, parent, options...)
}

// UpdateImageMemberRecord builds a fresh member request from the selected ID.
// The prior record supplies identity; its other attributes are not resubmitted.
func (c *Connection) UpdateImageMemberRecord(ctx context.Context, parent resource.Ref, input image.ImageMemberRecordRequest, options ...image.ImageMemberRecordWriteOption) (*image.ImageMemberRecord, error) {
	service, err := c.imageRecordService(ctx, "UpdateImageMemberRecord")
	if err != nil {
		return nil, err
	}
	return service.UpdateImageMemberRecord(ctx, parent, input, options...)
}

// RemoveImageMemberRecord deletes the fixed parent/member target and retains
// actual accepted response evidence. A clean missing response is ignored by default.
func (c *Connection) RemoveImageMemberRecord(ctx context.Context, parent resource.Ref, input image.ImageMemberRecordRequest, options ...image.RemoveImageMemberOption) (*image.ImageMemberAcknowledgement, error) {
	service, err := c.imageRecordService(ctx, "RemoveImageMemberRecord")
	if err != nil {
		return nil, err
	}
	return service.RemoveImageMemberRecord(ctx, parent, input, options...)
}

func (c *Connection) imageRecordService(ctx context.Context, operation string) (*image.Service, error) {
	return cloudimage.Service(ctx, c.cloudImageDependencies(), operation)
}
