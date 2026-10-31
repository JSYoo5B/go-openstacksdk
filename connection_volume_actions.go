package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"
)

// SetVolumeBootableStatus uses only cached Cinder and a required bool.
func (c *Connection) SetVolumeBootableStatus(ctx context.Context, input blockstorage.VolumeActionRequest, bootable bool) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.SetVolumeBootableStatus(ctx, c.cloudVolumeDependencies(), input, bootable)
}

// SetVolumeReadonly prepares originals once before selecting cached Cinder.
func (c *Connection) SetVolumeReadonly(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeReadonlyOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.SetVolumeReadonly(ctx, c.cloudVolumeDependencies(), input, options...)
}

// ReserveVolume selects cached Cinder only; no CurrentLocation is consumed.
func (c *Connection) ReserveVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.StateAction(ctx, c.cloudVolumeDependencies(), input, cinderaction.Reserve)
}

func (c *Connection) UnreserveVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.StateAction(ctx, c.cloudVolumeDependencies(), input, cinderaction.Unreserve)
}

func (c *Connection) BeginVolumeDetaching(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.StateAction(ctx, c.cloudVolumeDependencies(), input, cinderaction.BeginDetaching)
}

func (c *Connection) AbortVolumeDetaching(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.StateAction(ctx, c.cloudVolumeDependencies(), input, cinderaction.AbortDetaching)
}

func (c *Connection) ExtendVolume(ctx context.Context, input blockstorage.VolumeActionRequest, newSize int) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.ExtendVolume(ctx, c.cloudVolumeDependencies(), input, newSize)
}

// RetypeVolume prepares originals once before selecting cached Cinder.
func (c *Connection) RetypeVolume(ctx context.Context, input blockstorage.VolumeActionRequest, newType string, options ...blockstorage.VolumeRetypeOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.RetypeVolume(ctx, c.cloudVolumeDependencies(), input, newType, options...)
}

// CompleteVolumeExtend acknowledges one action without polling the volume.
func (c *Connection) CompleteVolumeExtend(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeExtendCompletionOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.CompleteVolumeExtend(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) ResetVolumeStatus(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeStatusResetOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.ResetVolumeStatus(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) MigrateVolume(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeMigrationOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.MigrateVolume(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) CompleteVolumeMigration(ctx context.Context, input blockstorage.VolumeActionRequest, newVolume string, options ...blockstorage.VolumeMigrationCompletionOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.CompleteVolumeMigration(ctx, c.cloudVolumeDependencies(), input, newVolume, options...)
}

// RevertVolumeToSnapshot selects cached Cinder after local input admission.
func (c *Connection) RevertVolumeToSnapshot(ctx context.Context, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.RevertVolumeToSnapshot(ctx, c.cloudVolumeDependencies(), input, snapshotID)
}

// UnmanageVolume uses cached Cinder and the SDK-owned null action policy.
func (c *Connection) UnmanageVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.UnmanageVolume(ctx, c.cloudVolumeDependencies(), input)
}
