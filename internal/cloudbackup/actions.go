package cloudbackup

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

type RestoreResult = cloudsnapshot.BackupRestoreResult
type ResetStatusResult = cloudsnapshot.BackupResetStatusResult

func Restore(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...cloudsnapshot.BackupRestoreOption) (*RestoreResult, error) {
	return cloudsnapshot.RestoreBackup(ctx, client, id, options...)
}
func ResetStatus(ctx context.Context, client *gophercloud.ServiceClient, id, status string) (*ResetStatusResult, error) {
	return cloudsnapshot.ResetBackupStatus(ctx, client, id, status)
}
