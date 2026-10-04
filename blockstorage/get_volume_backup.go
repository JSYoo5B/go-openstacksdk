package blockstorage

import (
	"context"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudbackup"
)

type VolumeBackupSelectionError = cloudbackup.SelectionError

// GetVolumeBackup uses exact member-first identity lookup for omitted/null
// filters. Every explicit nonnull filter uses complete search and JSON first
// selection; normal absence returns a nil logical Value and Backup.
func GetVolumeBackup(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumeBackupRequest, options ...VolumeBackupSearchOption) (*GetVolumeBackupResult, error) {
	return cloudbackup.Get(ctx, cinder, input.NameOrID, options...)
}
