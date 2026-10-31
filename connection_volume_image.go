package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"
)

// VolumeImageCreateRequest mirrors Cinder v3 Proxy create_image. Empty formats
// are Python-falsey: DiskFormat uses the cloud image_format setting and
// ContainerFormat uses "bare". AllowDuplicates is sent as the upload force flag.
// The pinned Proxy also accepts wait and timeout but never reads them.
type VolumeImageCreateRequest struct {
	Name            string
	VolumeID        string
	AllowDuplicates bool
	ContainerFormat string
	DiskFormat      string
}

// VolumeImageCreateResult keeps the Cinder upload evidence separately from the
// synchronized ImageRecord built from its image_id without a Glance request.
type VolumeImageCreateResult struct {
	Upload *blockstorage.VolumeImageUploadResult
	Image  *image.ImageRecord
}

// CreateVolumeImageRecord implements Cinder v3 Proxy create_image: one volume
// upload-to-image action, then Image.existing(id=image_id) with the current
// Connection location. It does not wait for the image or fetch it.
func (c *Connection) CreateVolumeImageRecord(ctx context.Context, input VolumeImageCreateRequest) (*VolumeImageCreateResult, error) {
	result, err := cloudvolume.CreateVolumeImageRecord(ctx, c.cloudVolumeDependencies(), cloudvolume.ImageCreateRequest(input))
	return (*VolumeImageCreateResult)(result), err
}

// SetVolumeImageMetadata prepares caller options once before selecting Cinder.
func (c *Connection) SetVolumeImageMetadata(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeImageMetadataOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.SetVolumeImageMetadata(ctx, c.cloudVolumeDependencies(), input, options...)
}

// DeleteVolumeImageMetadata uses the cached Cinder service and retains partial
// action acknowledgements. Omitted keys select the current image-metadata map.
func (c *Connection) DeleteVolumeImageMetadata(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeImageMetadataDeleteOption) (*blockstorage.VolumeImageMetadataDeleteResult, error) {
	return cloudvolume.DeleteVolumeImageMetadata(ctx, c.cloudVolumeDependencies(), input, options...)
}

// UploadVolumeToImage prepares originals once before selecting cached Cinder.
// It sends a volume action rather than uploading bytes through a Glance client.
func (c *Connection) UploadVolumeToImage(ctx context.Context, input blockstorage.VolumeActionRequest, imageName string, options ...blockstorage.VolumeImageUploadOption) (*blockstorage.VolumeImageUploadResult, error) {
	return cloudvolume.UploadVolumeToImage(ctx, c.cloudVolumeDependencies(), input, imageName, options...)
}
