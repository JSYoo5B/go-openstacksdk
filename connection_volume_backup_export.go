package gophercloudsdk

import (
	"context"
	"fmt"

	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/internal/cloudsnapshot"
	"gophercloudsdk/resource"
)

// ExportVolumeBackup validates one literal ID before selecting cached Cinder v3.
// Export returns opaque HTTP evidence and consumes no configured resource scope.
func (c *Connection) ExportVolumeBackup(ctx context.Context, input blockstorage.ExportVolumeBackupRequest) (*blockstorage.ExportVolumeBackupResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, "ExportVolumeBackup", err)
	}
	if c.provider == nil {
		return nil, wrapConnectionBackup(ctx, "ExportVolumeBackup", fmt.Errorf("%w: provider client is required", resource.ErrInvalidOption))
	}
	if err := cloudsnapshot.ValidateBackupExportID(input.BackupID); err != nil {
		return nil, wrapConnectionBackup(ctx, "ExportVolumeBackup", err)
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "ExportVolumeBackup", err)
	}
	if service == nil {
		return nil, wrapConnectionBackup(ctx, "ExportVolumeBackup", fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, "ExportVolumeBackup", err)
	}
	return blockstorage.ExportVolumeBackup(ctx, service.RawClient(), input)
}
