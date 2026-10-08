package blockstorage

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudbackup"
	"github.com/gophercloud/gophercloud/v2"
)

// SearchVolumeBackups completes a queryless detailed list before applying
// exact/glob identity, recursive mapping or a JSON expression. Projection has
// no invented association with a raw server resource.
func SearchVolumeBackups(ctx context.Context, cinder *gophercloud.ServiceClient, input SearchVolumeBackupsRequest, options ...VolumeBackupSearchOption) (*SearchVolumeBackupsResult, error) {
	return cloudbackup.Search(ctx, cinder, input.NameOrID, options...)
}
