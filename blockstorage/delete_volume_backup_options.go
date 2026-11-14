package blockstorage

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type DeleteVolumeBackupRequest struct{ NameOrID string }
type DeleteVolumeBackupOpts = cloudsnapshot.BackupDeleteOptions
type DeleteVolumeBackupOption = cloudsnapshot.BackupDeleteOption

func WithDeleteVolumeBackupOptions(value DeleteVolumeBackupOpts) DeleteVolumeBackupOption {
	return cloudsnapshot.WithBackupDeleteOptions(value)
}
func WithDeleteVolumeBackupForce(value bool) DeleteVolumeBackupOption {
	return cloudsnapshot.WithBackupDeleteForce(value)
}
func WithDeleteVolumeBackupWait(value bool) DeleteVolumeBackupOption {
	return cloudsnapshot.WithBackupDeleteWait(value)
}

// WithDeleteVolumeBackupWaitOptions appends shared wait options for the backup deletion wait;
// repeated calls append and the later option wins per control. The wait
// defaults to unlimited with two-second polling and honors resource.WithTimeout,
// resource.WithUnlimitedWait, resource.WithPollInterval and
// resource.WithWaitStrategy. A zero or negative timeout or interval and other
// controls fail with resource.ErrInvalidOption before any request.
func WithDeleteVolumeBackupWaitOptions(options ...resource.WaitOption) DeleteVolumeBackupOption {
	return cloudsnapshot.WithBackupDeleteWaitOptions(options...)
}

func PrepareDeleteVolumeBackupOptions(ctx context.Context, options ...DeleteVolumeBackupOption) (DeleteVolumeBackupOpts, error) {
	return cloudsnapshot.PrepareBackupDelete(ctx, options...)
}
