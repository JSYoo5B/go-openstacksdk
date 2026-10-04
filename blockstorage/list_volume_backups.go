package blockstorage

import (
	"context"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudbackup"
)

// ListVolumeBackups returns the complete owned cloud view. Detailed and
// pagination default to true; typed controls override raw dictionary controls.
func ListVolumeBackups(ctx context.Context, cinder *gophercloud.ServiceClient, options ...VolumeBackupListOption) (*ListVolumeBackupsResult, error) {
	return cloudbackup.List(ctx, cinder, options...)
}
