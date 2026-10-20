package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
)

// AttachCinderVolume prepares originals once and uses cached Cinder directly.
func (c *Connection) AttachCinderVolume(ctx context.Context, input blockstorage.VolumeActionRequest, mountpoint string, options ...blockstorage.CinderVolumeAttachOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "AttachCinderVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareCinderVolumeAttachOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateMountpoint(mountpoint)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.DirectAttach(ctx, client, input.VolumeID, mountpoint, cinderaction.WithDirectAttachOptions(policy))
}

// DetachCinderVolume performs one direct action without a Nova detach workflow.
func (c *Connection) DetachCinderVolume(ctx context.Context, input blockstorage.VolumeActionRequest, attachmentID string, options ...blockstorage.CinderVolumeDetachOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "DetachCinderVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateAttachmentID(attachmentID)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.DirectDetach(ctx, client, input.VolumeID, attachmentID, cinderaction.WithDirectDetachOptions(policy))
}
