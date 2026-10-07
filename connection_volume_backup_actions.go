package gophercloudsdk

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// RestoreVolumeBackup owns original options before location/Cinder selection.
func (c *Connection) RestoreVolumeBackup(ctx context.Context, input blockstorage.RestoreVolumeBackupRequest, options ...blockstorage.RestoreVolumeBackupOption) (*blockstorage.RestoreVolumeBackupResult, error) {
	const operation = "RestoreVolumeBackup"
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareRestoreVolumeBackupOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateBackupRestoreInput(input.BackupID, policy)
	}
	if err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	client, err := c.cinderCloudReadClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	return blockstorage.RestoreVolumeBackup(ctx, client, input, blockstorage.WithRestoreVolumeBackupOptions(policy))
}

// ResetVolumeBackupStatus selects cached Cinder only and forwards opaque status.
func (c *Connection) ResetVolumeBackupStatus(ctx context.Context, input blockstorage.ResetVolumeBackupStatusRequest) (*blockstorage.ResetVolumeBackupStatusResult, error) {
	const operation = "ResetVolumeBackupStatus"
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	if c.provider == nil {
		return nil, wrapConnectionBackup(ctx, operation, fmt.Errorf("%w: provider client is required", resource.ErrInvalidOption))
	}
	if err := cloudsnapshot.ValidateBackupResetStatusInput(input.BackupID, input.Status); err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	if service == nil {
		return nil, wrapConnectionBackup(ctx, operation, fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	return blockstorage.ResetVolumeBackupStatus(ctx, service.RawClient(), input)
}
