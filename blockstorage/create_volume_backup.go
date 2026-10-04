package blockstorage

import (
	"context"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudbackup"
)

// CreateVolumeBackup submits the fixed six-field cloud body with literal IDs.
// It defaults to force=false, incremental=false and wait=true. A nil timeout
// is unlimited; a cached available resource completes without a fetch.
func CreateVolumeBackup(ctx context.Context, client *gophercloud.ServiceClient, input CreateVolumeBackupRequest, options ...CreateVolumeBackupOption) (*CreateVolumeBackupResult, error) {
	return cloudbackup.Create(ctx, client, input.VolumeID, options...)
}
