package blockstorage

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudbackup"
	"github.com/gophercloud/gophercloud/v2"
)

// DeleteVolumeBackup resolves exactly one Backup and defaults to wait=false.
// Force sends the source action's null payload at microversion3.64. Empty input
// and successful absence return false; rejected mutations remain errors.
func DeleteVolumeBackup(ctx context.Context, client *gophercloud.ServiceClient, input DeleteVolumeBackupRequest, options ...DeleteVolumeBackupOption) (*DeleteVolumeBackupResult, error) {
	return cloudbackup.Delete(ctx, client, input.NameOrID, options...)
}
