package gophercloudsdk

import (
	"context"

	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/internal/cloudsnapshot"
)

// CreateVolumeBackup prepares original options and validates literal input
// before selecting cached Cinder v3. Location and defaults are SDK-owned.
func (c *Connection) CreateVolumeBackup(ctx context.Context, input blockstorage.CreateVolumeBackupRequest, options ...blockstorage.CreateVolumeBackupOption) (*blockstorage.CreateVolumeBackupResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, "CreateVolumeBackup", err)
	}
	policy, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateBackupCreateInput(input.VolumeID, policy)
	}
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "CreateVolumeBackup", err)
	}
	client, err := c.cinderCloudReadClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "CreateVolumeBackup", err)
	}
	return blockstorage.CreateVolumeBackup(ctx, client, input, blockstorage.WithCreateVolumeBackupOptions(policy))
}

// DeleteVolumeBackup skips service/location selection for an empty input.
// Original callbacks still run once; a nil Connection is sufficient for this
// service-free no-op. Nonempty input resolves and deletes through cached v3.
func (c *Connection) DeleteVolumeBackup(ctx context.Context, input blockstorage.DeleteVolumeBackupRequest, options ...blockstorage.DeleteVolumeBackupOption) (*blockstorage.DeleteVolumeBackupResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, "DeleteVolumeBackup", err)
	}
	if input.NameOrID != "" {
		if err := c.volumeMutationPreflight(ctx); err != nil {
			return nil, wrapConnectionBackup(ctx, "DeleteVolumeBackup", err)
		}
	}
	policy, err := blockstorage.PrepareDeleteVolumeBackupOptions(ctx, options...)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "DeleteVolumeBackup", err)
	}
	if input.NameOrID == "" {
		return blockstorage.DeleteVolumeBackup(ctx, nil, input, blockstorage.WithDeleteVolumeBackupOptions(policy))
	}
	client, err := c.cinderCloudReadClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "DeleteVolumeBackup", err)
	}
	return blockstorage.DeleteVolumeBackup(ctx, client, input, blockstorage.WithDeleteVolumeBackupOptions(policy))
}
