package blockstorage

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudbackup"
)

type ResetVolumeBackupStatusRequest struct{ BackupID, Status string }
type ResetVolumeBackupStatusResult = cloudbackup.ResetStatusResult

func ResetVolumeBackupStatus(ctx context.Context, client *gophercloud.ServiceClient, input ResetVolumeBackupStatusRequest) (*ResetVolumeBackupStatusResult, error) {
	return cloudbackup.ResetStatus(ctx, client, input.BackupID, input.Status)
}
