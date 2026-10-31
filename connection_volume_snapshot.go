package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"
)

// ListVolumeSnapshots owns original options and location before selecting the
// connection's cached Cinder v3 service. SDK callers need no custom builder.
func (c *Connection) ListVolumeSnapshots(ctx context.Context, options ...blockstorage.VolumeSnapshotListOption) (*blockstorage.ListVolumeSnapshotsResult, error) {
	return cloudvolume.ListVolumeSnapshots(ctx, c.cloudVolumeDependencies(), options...)
}

func (c *Connection) SearchVolumeSnapshots(ctx context.Context, input blockstorage.SearchVolumeSnapshotsRequest, options ...blockstorage.VolumeSnapshotSearchOption) (*blockstorage.SearchVolumeSnapshotsResult, error) {
	return cloudvolume.SearchVolumeSnapshots(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolumeSnapshot(ctx context.Context, input blockstorage.GetVolumeSnapshotRequest, options ...blockstorage.VolumeSnapshotSearchOption) (*blockstorage.GetVolumeSnapshotResult, error) {
	return cloudvolume.GetVolumeSnapshot(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolumeSnapshotByID(ctx context.Context, input blockstorage.GetVolumeSnapshotByIDRequest, options ...blockstorage.VolumeSnapshotReadOption) (*blockstorage.GetVolumeSnapshotResult, error) {
	return cloudvolume.GetVolumeSnapshotByID(ctx, c.cloudVolumeDependencies(), input, options...)
}

// CreateVolumeSnapshot prepares original options and validates literal input
// before selecting cached Cinder v3. Location and defaults are SDK-owned.
func (c *Connection) CreateVolumeSnapshot(ctx context.Context, input blockstorage.CreateVolumeSnapshotRequest, options ...blockstorage.CreateVolumeSnapshotOption) (*blockstorage.CreateVolumeSnapshotResult, error) {
	return cloudvolume.CreateVolumeSnapshot(ctx, c.cloudVolumeDependencies(), input, options...)
}

// DeleteVolumeSnapshot skips service/location selection for an empty input.
// Original callbacks still run once; a nil Connection is sufficient for this
// service-free no-op. Nonempty input resolves and deletes through cached v3.
func (c *Connection) DeleteVolumeSnapshot(ctx context.Context, input blockstorage.DeleteVolumeSnapshotRequest, options ...blockstorage.DeleteVolumeSnapshotOption) (*blockstorage.DeleteVolumeSnapshotResult, error) {
	return cloudvolume.DeleteVolumeSnapshot(ctx, c.cloudVolumeDependencies(), input, options...)
}
