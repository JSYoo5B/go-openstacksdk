package blockstorage

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudbackup"
)

type RestoreVolumeBackupResult = cloudbackup.RestoreResult

func RestoreVolumeBackup(ctx context.Context, client *gophercloud.ServiceClient, input RestoreVolumeBackupRequest, options ...RestoreVolumeBackupOption) (*RestoreVolumeBackupResult, error) {
	return cloudbackup.Restore(ctx, client, input.BackupID, options...)
}
