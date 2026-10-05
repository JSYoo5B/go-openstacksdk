package gophercloudsdk

import (
	"context"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/cinderaction"
)

// SetVolumeImageMetadata prepares caller options once before selecting Cinder.
func (c *Connection) SetVolumeImageMetadata(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeImageMetadataOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "SetVolumeImageMetadata"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.SetImageMetadata(ctx, client, input.VolumeID, cinderaction.WithImageMetadataOptions(policy))
}

// DeleteVolumeImageMetadata uses the cached Cinder service and retains partial
// action acknowledgements. Omitted keys select the current image-metadata map.
func (c *Connection) DeleteVolumeImageMetadata(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeImageMetadataDeleteOption) (*blockstorage.VolumeImageMetadataDeleteResult, error) {
	const operation = "DeleteVolumeImageMetadata"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.DeleteImageMetadata(ctx, client, input.VolumeID, cinderaction.WithImageMetadataDeleteOptions(policy))
}
