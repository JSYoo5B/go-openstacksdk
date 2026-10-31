package cloudvolume

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

func wrapBackup(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volume backup", cloudread.ContextError(ctx, err))
}

// ListVolumeBackups prepares owned policy before selecting cached Cinder v3.
func ListVolumeBackups(ctx context.Context, deps Dependencies, options ...blockstorage.VolumeBackupListOption) (*blockstorage.ListVolumeBackupsResult, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapBackup(ctx, "ListVolumeBackups", err)
	}
	policy, err := blockstorage.PrepareVolumeBackupListOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateBackupListOptions(policy)
	}
	if err != nil {
		return nil, wrapBackup(ctx, "ListVolumeBackups", err)
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	if err != nil {
		return nil, wrapBackup(ctx, "ListVolumeBackups", err)
	}
	return blockstorage.ListVolumeBackups(ctx, client, blockstorage.WithVolumeBackupListOptions(policy))
}

// SearchVolumeBackups prepares owned policy before selecting cached Cinder v3.
func SearchVolumeBackups(ctx context.Context, deps Dependencies, input blockstorage.SearchVolumeBackupsRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.SearchVolumeBackupsResult, error) {
	client, policy, err := prepareBackupSearch(ctx, deps, options)
	if err != nil {
		return nil, wrapBackup(ctx, "SearchVolumeBackups", err)
	}
	return blockstorage.SearchVolumeBackups(ctx, client, input, blockstorage.WithVolumeBackupSearchOptions(policy))
}

// GetVolumeBackup prepares owned policy before selecting cached Cinder v3.
func GetVolumeBackup(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeBackupRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.GetVolumeBackupResult, error) {
	client, policy, err := prepareBackupSearch(ctx, deps, options)
	if err != nil {
		return nil, wrapBackup(ctx, "GetVolumeBackup", err)
	}
	return blockstorage.GetVolumeBackup(ctx, client, input, blockstorage.WithVolumeBackupSearchOptions(policy))
}

func prepareBackupSearch(ctx context.Context, deps Dependencies, options []blockstorage.VolumeBackupSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeBackupSearchOpts, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, blockstorage.VolumeBackupSearchOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeBackupSearchOptions(ctx, options...)
	if err != nil {
		return nil, blockstorage.VolumeBackupSearchOpts{}, err
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	return client, policy, err
}

// CreateVolumeBackup prepares original options and validates literal input
// before selecting cached Cinder v3.
func CreateVolumeBackup(ctx context.Context, deps Dependencies, input blockstorage.CreateVolumeBackupRequest, options ...blockstorage.CreateVolumeBackupOption) (*blockstorage.CreateVolumeBackupResult, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapBackup(ctx, "CreateVolumeBackup", err)
	}
	policy, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateBackupCreateInput(input.VolumeID, policy)
	}
	if err != nil {
		return nil, wrapBackup(ctx, "CreateVolumeBackup", err)
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	if err != nil {
		return nil, wrapBackup(ctx, "CreateVolumeBackup", err)
	}
	return blockstorage.CreateVolumeBackup(ctx, client, input, blockstorage.WithCreateVolumeBackupOptions(policy))
}

// DeleteVolumeBackup skips service/location selection for an empty input;
// the zero Dependencies are sufficient for this service-free no-op.
func DeleteVolumeBackup(ctx context.Context, deps Dependencies, input blockstorage.DeleteVolumeBackupRequest, options ...blockstorage.DeleteVolumeBackupOption) (*blockstorage.DeleteVolumeBackupResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapBackup(ctx, "DeleteVolumeBackup", err)
	}
	if input.NameOrID != "" {
		if err := mutationPreflight(ctx, deps); err != nil {
			return nil, wrapBackup(ctx, "DeleteVolumeBackup", err)
		}
	}
	policy, err := blockstorage.PrepareDeleteVolumeBackupOptions(ctx, options...)
	if err != nil {
		return nil, wrapBackup(ctx, "DeleteVolumeBackup", err)
	}
	if input.NameOrID == "" {
		return blockstorage.DeleteVolumeBackup(ctx, nil, input, blockstorage.WithDeleteVolumeBackupOptions(policy))
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	if err != nil {
		return nil, wrapBackup(ctx, "DeleteVolumeBackup", err)
	}
	return blockstorage.DeleteVolumeBackup(ctx, client, input, blockstorage.WithDeleteVolumeBackupOptions(policy))
}

// RestoreVolumeBackup owns original options before location/Cinder selection.
func RestoreVolumeBackup(ctx context.Context, deps Dependencies, input blockstorage.RestoreVolumeBackupRequest, options ...blockstorage.RestoreVolumeBackupOption) (*blockstorage.RestoreVolumeBackupResult, error) {
	const operation = "RestoreVolumeBackup"
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapBackup(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareRestoreVolumeBackupOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateBackupRestoreInput(input.BackupID, policy)
	}
	if err != nil {
		return nil, wrapBackup(ctx, operation, err)
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	if err != nil {
		return nil, wrapBackup(ctx, operation, err)
	}
	return blockstorage.RestoreVolumeBackup(ctx, client, input, blockstorage.WithRestoreVolumeBackupOptions(policy))
}

// backupActionClient runs the provider-requiring preflight, the literal input
// check and the cached Cinder v3 selection shared by the location-free backup
// actions.
func backupActionClient(ctx context.Context, deps Dependencies, validate func() error) (*gophercloud.ServiceClient, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, err
	}
	if deps.Provider() == nil {
		return nil, fmt.Errorf("%w: provider client is required", resource.ErrInvalidOption)
	}
	if err := validate(); err != nil {
		return nil, err
	}
	service, err := deps.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption)
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	return service.RawClient(), nil
}

// ResetVolumeBackupStatus selects cached Cinder only and forwards opaque status.
func ResetVolumeBackupStatus(ctx context.Context, deps Dependencies, input blockstorage.ResetVolumeBackupStatusRequest) (*blockstorage.ResetVolumeBackupStatusResult, error) {
	client, err := backupActionClient(ctx, deps, func() error {
		return cloudsnapshot.ValidateBackupResetStatusInput(input.BackupID, input.Status)
	})
	if err != nil {
		return nil, wrapBackup(ctx, "ResetVolumeBackupStatus", err)
	}
	return blockstorage.ResetVolumeBackupStatus(ctx, client, input)
}

// ExportVolumeBackup validates one literal ID before selecting cached Cinder v3.
func ExportVolumeBackup(ctx context.Context, deps Dependencies, input blockstorage.ExportVolumeBackupRequest) (*blockstorage.ExportVolumeBackupResult, error) {
	client, err := backupActionClient(ctx, deps, func() error {
		return cloudsnapshot.ValidateBackupExportID(input.BackupID)
	})
	if err != nil {
		return nil, wrapBackup(ctx, "ExportVolumeBackup", err)
	}
	return blockstorage.ExportVolumeBackup(ctx, client, input)
}

// ExportVolumeBackupRecord selects only cached Cinder v3 after local preflight.
func ExportVolumeBackupRecord(ctx context.Context, deps Dependencies, input blockstorage.ExportVolumeBackupRecordRequest) (*blockstorage.ExportVolumeBackupRecordResult, error) {
	client, err := backupActionClient(ctx, deps, func() error {
		return cloudsnapshot.ValidateBackupExportID(input.BackupID)
	})
	if err != nil {
		return nil, wrapBackup(ctx, "ExportVolumeBackupRecord", err)
	}
	return blockstorage.ExportVolumeBackupRecord(ctx, client, input)
}

// ImportVolumeBackup uses cached Cinder without consuming the current location.
func ImportVolumeBackup(ctx context.Context, deps Dependencies, input blockstorage.ImportVolumeBackupRequest) (*blockstorage.ImportVolumeBackupResult, error) {
	client, err := backupActionClient(ctx, deps, func() error {
		return cloudsnapshot.ValidateBackupImportInput(input.BackupService, input.BackupURL)
	})
	if err != nil {
		return nil, wrapBackup(ctx, "ImportVolumeBackup", err)
	}
	return blockstorage.ImportVolumeBackup(ctx, client, input)
}
