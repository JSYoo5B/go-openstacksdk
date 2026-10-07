package blockstorage

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudbackup"
	"github.com/gophercloud/gophercloud/v2"
)

type ImportVolumeBackupRequest struct {
	BackupService string
	BackupURL     string
}

type ImportVolumeBackupResult = cloudbackup.ImportResult

// ImportVolumeBackup imports an opaque record and returns an owned Backup view.
func ImportVolumeBackup(ctx context.Context, client *gophercloud.ServiceClient, input ImportVolumeBackupRequest) (*ImportVolumeBackupResult, error) {
	return cloudbackup.Import(ctx, client, input.BackupService, input.BackupURL)
}
