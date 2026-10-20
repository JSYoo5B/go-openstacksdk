package blockstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudbackup"
	"github.com/gophercloud/gophercloud/v2"
)

type ResetVolumeBackupStatusRequest struct{ BackupID, Status string }
type ResetVolumeBackupStatusResult = cloudbackup.ResetStatusResult

func ResetVolumeBackupStatus(ctx context.Context, client *gophercloud.ServiceClient, input ResetVolumeBackupStatusRequest) (*ResetVolumeBackupStatusResult, error) {
	return cloudbackup.ResetStatus(ctx, client, input.BackupID, input.Status)
}
