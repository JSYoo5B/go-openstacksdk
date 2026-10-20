package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImportVolumeBackup uses cached Cinder without consuming CurrentLocation.
func (c *Connection) ImportVolumeBackup(ctx context.Context, input blockstorage.ImportVolumeBackupRequest) (*blockstorage.ImportVolumeBackupResult, error) {
	const operation = "ImportVolumeBackup"
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	if c.provider == nil {
		return nil, wrapConnectionBackup(ctx, operation, fmt.Errorf("%w: provider client is required", resource.ErrInvalidOption))
	}
	if err := cloudsnapshot.ValidateBackupImportInput(input.BackupService, input.BackupURL); err != nil {
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
	return blockstorage.ImportVolumeBackup(ctx, service.RawClient(), input)
}
