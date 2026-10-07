package gophercloudsdk

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/cinderaction"
)

// UploadVolumeToImage prepares originals once before selecting cached Cinder.
// It sends a volume action rather than uploading bytes through a Glance client.
func (c *Connection) UploadVolumeToImage(ctx context.Context, input blockstorage.VolumeActionRequest, imageName string, options ...blockstorage.VolumeImageUploadOption) (*blockstorage.VolumeImageUploadResult, error) {
	const operation = "UploadVolumeToImage"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeImageUploadOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateImageName(imageName)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.UploadToImage(ctx, client, input.VolumeID, imageName, cinderaction.WithImageUploadOptions(policy))
}
