package gophercloudsdk

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/cinderaction"
)

func (c *Connection) ExtendVolume(ctx context.Context, input blockstorage.VolumeActionRequest, newSize int) (*blockstorage.VolumeActionResult, error) {
	const operation = "ExtendVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Extend(ctx, client, input.VolumeID, newSize)
}

// RetypeVolume prepares originals once before selecting cached Cinder.
func (c *Connection) RetypeVolume(ctx context.Context, input blockstorage.VolumeActionRequest, newType string, options ...blockstorage.VolumeRetypeOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "RetypeVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeRetypeOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateNewType(newType)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Retype(ctx, client, input.VolumeID, newType, cinderaction.WithRetypeOptions(policy))
}

// CompleteVolumeExtend acknowledges one action without polling the volume.
func (c *Connection) CompleteVolumeExtend(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeExtendCompletionOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "CompleteVolumeExtend"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeExtendCompletionOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.CompleteExtend(ctx, client, input.VolumeID, cinderaction.WithExtendCompletionOptions(policy))
}
