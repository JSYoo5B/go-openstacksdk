package gophercloudsdk

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
)

// CreateVolumeSnapshot prepares original options and validates literal input
// before selecting cached Cinder v3. Location and defaults are SDK-owned.
func (c *Connection) CreateVolumeSnapshot(ctx context.Context, input blockstorage.CreateVolumeSnapshotRequest, options ...blockstorage.CreateVolumeSnapshotOption) (*blockstorage.CreateVolumeSnapshotResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionSnapshot(ctx, "CreateVolumeSnapshot", err)
	}
	policy, err := blockstorage.PrepareCreateVolumeSnapshotOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateCreateInput(input.VolumeID, policy)
	}
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "CreateVolumeSnapshot", err)
	}
	client, err := c.snapshotClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "CreateVolumeSnapshot", err)
	}
	return blockstorage.CreateVolumeSnapshot(ctx, client, input, blockstorage.WithCreateVolumeSnapshotOptions(policy))
}

// DeleteVolumeSnapshot skips service/location selection for an empty input.
// Original callbacks still run once; a nil Connection is sufficient for this
// service-free no-op. Nonempty input resolves and deletes through cached v3.
func (c *Connection) DeleteVolumeSnapshot(ctx context.Context, input blockstorage.DeleteVolumeSnapshotRequest, options ...blockstorage.DeleteVolumeSnapshotOption) (*blockstorage.DeleteVolumeSnapshotResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapConnectionSnapshot(ctx, "DeleteVolumeSnapshot", err)
	}
	if input.NameOrID != "" {
		if err := c.volumeMutationPreflight(ctx); err != nil {
			return nil, wrapConnectionSnapshot(ctx, "DeleteVolumeSnapshot", err)
		}
	}
	policy, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(ctx, options...)
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "DeleteVolumeSnapshot", err)
	}
	if input.NameOrID == "" {
		return blockstorage.DeleteVolumeSnapshot(ctx, nil, input, blockstorage.WithDeleteVolumeSnapshotOptions(policy))
	}
	client, err := c.snapshotClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "DeleteVolumeSnapshot", err)
	}
	return blockstorage.DeleteVolumeSnapshot(ctx, client, input, blockstorage.WithDeleteVolumeSnapshotOptions(policy))
}
