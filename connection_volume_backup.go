package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"
)

// ListVolumeBackups prepares owned policy before selecting cached Cinder v3.
func (c *Connection) ListVolumeBackups(ctx context.Context, options ...blockstorage.VolumeBackupListOption) (*blockstorage.ListVolumeBackupsResult, error) {
	return cloudvolume.ListVolumeBackups(ctx, c.cloudVolumeDependencies(), options...)
}

func (c *Connection) SearchVolumeBackups(ctx context.Context, input blockstorage.SearchVolumeBackupsRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.SearchVolumeBackupsResult, error) {
	return cloudvolume.SearchVolumeBackups(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolumeBackup(ctx context.Context, input blockstorage.GetVolumeBackupRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.GetVolumeBackupResult, error) {
	return cloudvolume.GetVolumeBackup(ctx, c.cloudVolumeDependencies(), input, options...)
}

// CreateVolumeBackup prepares original options and validates literal input
// before selecting cached Cinder v3. Location and defaults are SDK-owned.
func (c *Connection) CreateVolumeBackup(ctx context.Context, input blockstorage.CreateVolumeBackupRequest, options ...blockstorage.CreateVolumeBackupOption) (*blockstorage.CreateVolumeBackupResult, error) {
	return cloudvolume.CreateVolumeBackup(ctx, c.cloudVolumeDependencies(), input, options...)
}

// DeleteVolumeBackup skips service/location selection for an empty input.
// Original callbacks still run once; a nil Connection is sufficient for this
// service-free no-op. Nonempty input resolves and deletes through cached v3.
func (c *Connection) DeleteVolumeBackup(ctx context.Context, input blockstorage.DeleteVolumeBackupRequest, options ...blockstorage.DeleteVolumeBackupOption) (*blockstorage.DeleteVolumeBackupResult, error) {
	return cloudvolume.DeleteVolumeBackup(ctx, c.cloudVolumeDependencies(), input, options...)
}

// RestoreVolumeBackup owns original options before location/Cinder selection.
func (c *Connection) RestoreVolumeBackup(ctx context.Context, input blockstorage.RestoreVolumeBackupRequest, options ...blockstorage.RestoreVolumeBackupOption) (*blockstorage.RestoreVolumeBackupResult, error) {
	return cloudvolume.RestoreVolumeBackup(ctx, c.cloudVolumeDependencies(), input, options...)
}

// ResetVolumeBackupStatus selects cached Cinder only and forwards opaque status.
func (c *Connection) ResetVolumeBackupStatus(ctx context.Context, input blockstorage.ResetVolumeBackupStatusRequest) (*blockstorage.ResetVolumeBackupStatusResult, error) {
	return cloudvolume.ResetVolumeBackupStatus(ctx, c.cloudVolumeDependencies(), input)
}

// ExportVolumeBackup validates one literal ID before selecting cached Cinder v3.
// Export returns opaque HTTP evidence and consumes no configured resource scope.
func (c *Connection) ExportVolumeBackup(ctx context.Context, input blockstorage.ExportVolumeBackupRequest) (*blockstorage.ExportVolumeBackupResult, error) {
	return cloudvolume.ExportVolumeBackup(ctx, c.cloudVolumeDependencies(), input)
}

// ExportVolumeBackupRecord selects only cached Cinder v3 after local preflight.
func (c *Connection) ExportVolumeBackupRecord(ctx context.Context, input blockstorage.ExportVolumeBackupRecordRequest) (*blockstorage.ExportVolumeBackupRecordResult, error) {
	return cloudvolume.ExportVolumeBackupRecord(ctx, c.cloudVolumeDependencies(), input)
}

// ImportVolumeBackup uses cached Cinder without consuming CurrentLocation.
func (c *Connection) ImportVolumeBackup(ctx context.Context, input blockstorage.ImportVolumeBackupRequest) (*blockstorage.ImportVolumeBackupResult, error) {
	return cloudvolume.ImportVolumeBackup(ctx, c.cloudVolumeDependencies(), input)
}
