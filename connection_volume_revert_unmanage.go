package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
)

// RevertVolumeToSnapshot selects cached Cinder after local input admission.
func (c *Connection) RevertVolumeToSnapshot(ctx context.Context, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
	const operation = "RevertVolumeToSnapshot"
	err := c.volumeFlagPreflight(ctx)
	if err == nil {
		err = cinderaction.ValidateSnapshotID(snapshotID)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.RevertToSnapshot(ctx, client, input.VolumeID, snapshotID)
}

// UnmanageVolume uses cached Cinder and the SDK-owned null action policy.
func (c *Connection) UnmanageVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	const operation = "UnmanageVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Unmanage(ctx, client, input.VolumeID)
}
