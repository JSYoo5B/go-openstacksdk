package openstack

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudimage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"
	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"

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
