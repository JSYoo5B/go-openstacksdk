package blockstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudbackup"
	"github.com/gophercloud/gophercloud/v2"
)

type RestoreVolumeBackupResult = cloudbackup.RestoreResult

func RestoreVolumeBackup(ctx context.Context, client *gophercloud.ServiceClient, input RestoreVolumeBackupRequest, options ...RestoreVolumeBackupOption) (*RestoreVolumeBackupResult, error) {
	return cloudbackup.Restore(ctx, client, input.BackupID, options...)
}
