package backups

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudbackup"
	"gophercloudsdk/internal/cloudsnapshot"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type RestoreBackupOpts = cloudsnapshot.BackupRestoreOptions
type RestoreBackupOption = cloudsnapshot.BackupRestoreOption
type RestoreBackupResult = cloudbackup.RestoreResult
type ResetBackupStatusResult = cloudbackup.ResetStatusResult

func WithRestoreBackupOptions(value RestoreBackupOpts) RestoreBackupOption {
	return cloudsnapshot.WithBackupRestoreOptions(value)
}
func WithRestoreBackupVolumeID(value string) RestoreBackupOption {
	return cloudsnapshot.WithBackupRestoreVolumeID(value)
}
func WithRestoreBackupName(value string) RestoreBackupOption {
	return cloudsnapshot.WithBackupRestoreName(value)
}
func WithRestoreBackupSeed(value json.RawMessage) RestoreBackupOption {
	return cloudsnapshot.WithBackupRestoreSeed(value)
}
func WithRestoreBackupLocation(value resource.CloudLocation) RestoreBackupOption {
	return cloudsnapshot.WithBackupRestoreLocation(value)
}
func PrepareRestoreBackupOptions(ctx context.Context, options ...RestoreBackupOption) (RestoreBackupOpts, error) {
	return cloudsnapshot.PrepareBackupRestore(ctx, options...)
}

func (a *API) RestoreBackup(ctx context.Context, id string, options ...RestoreBackupOption) (*RestoreBackupResult, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	result, err := cloudbackup.Restore(ctx, client, id, options...)
	return result, request.Wrap("RestoreBackup", "backups", err)
}
func (a *API) ResetBackupStatus(ctx context.Context, id, status string) (*ResetBackupStatusResult, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	result, err := cloudbackup.ResetStatus(ctx, client, id, status)
	return result, request.Wrap("ResetBackupStatus", "backups", err)
}
