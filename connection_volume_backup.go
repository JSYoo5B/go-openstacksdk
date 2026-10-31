package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// ListVolumeBackups prepares owned policy before selecting cached Cinder v3.
func (c *Connection) ListVolumeBackups(ctx context.Context, options ...blockstorage.VolumeBackupListOption) (*blockstorage.ListVolumeBackupsResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, "ListVolumeBackups", err)
	}
	policy, err := blockstorage.PrepareVolumeBackupListOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateBackupListOptions(policy)
	}
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "ListVolumeBackups", err)
	}
	client, err := c.cinderCloudReadClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "ListVolumeBackups", err)
	}
	return blockstorage.ListVolumeBackups(ctx, client, blockstorage.WithVolumeBackupListOptions(policy))
}

func (c *Connection) SearchVolumeBackups(ctx context.Context, input blockstorage.SearchVolumeBackupsRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.SearchVolumeBackupsResult, error) {
	client, policy, err := c.prepareBackupSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "SearchVolumeBackups", err)
	}
	return blockstorage.SearchVolumeBackups(ctx, client, input, blockstorage.WithVolumeBackupSearchOptions(policy))
}

func (c *Connection) GetVolumeBackup(ctx context.Context, input blockstorage.GetVolumeBackupRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.GetVolumeBackupResult, error) {
	client, policy, err := c.prepareBackupSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "GetVolumeBackup", err)
	}
	return blockstorage.GetVolumeBackup(ctx, client, input, blockstorage.WithVolumeBackupSearchOptions(policy))
}

func (c *Connection) prepareBackupSearch(ctx context.Context, options []blockstorage.VolumeBackupSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeBackupSearchOpts, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, blockstorage.VolumeBackupSearchOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeBackupSearchOptions(ctx, options...)
	if err != nil {
		return nil, blockstorage.VolumeBackupSearchOpts{}, err
	}
	client, err := c.cinderCloudReadClient(ctx, &policy.Location)
	return client, policy, err
}

func wrapConnectionBackup(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volume backup", cloudread.ContextError(ctx, err))
}

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

// ExportVolumeBackupRecord selects only cached Cinder v3 after local preflight.
func (c *Connection) ExportVolumeBackupRecord(ctx context.Context, input blockstorage.ExportVolumeBackupRecordRequest) (*blockstorage.ExportVolumeBackupRecordResult, error) {
	const operation = "ExportVolumeBackupRecord"
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, operation, err)
	}
	if c.provider == nil {
		return nil, wrapConnectionBackup(ctx, operation, fmt.Errorf("%w: provider client is required", resource.ErrInvalidOption))
	}
	if err := cloudsnapshot.ValidateBackupExportID(input.BackupID); err != nil {
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
	return blockstorage.ExportVolumeBackupRecord(ctx, service.RawClient(), input)
}

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
