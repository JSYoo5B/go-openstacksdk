package blockstorage

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type CreateVolumeBackupRequest struct{ VolumeID string }
type CreateVolumeBackupOpts = cloudsnapshot.BackupCreateOptions
type CreateVolumeBackupOption = cloudsnapshot.BackupCreateOption

func WithCreateVolumeBackupOptions(value CreateVolumeBackupOpts) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateOptions(value)
}
func WithCreateVolumeBackupName(value string) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateName(value)
}
func WithCreateVolumeBackupDescription(value string) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateDescription(value)
}
func WithCreateVolumeBackupSnapshotID(value string) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateSnapshotID(value)
}
func WithCreateVolumeBackupForce(value bool) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateForce(value)
}
func WithCreateVolumeBackupIncremental(value bool) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateIncremental(value)
}
func WithCreateVolumeBackupWait(value bool) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateWait(value)
}

// WithCreateVolumeBackupWaitOptions appends shared wait options for the backup creation wait;
// repeated calls append and the later option wins per control. The wait
// defaults to unlimited with two-second polling and honors resource.WithTimeout,
// resource.WithUnlimitedWait, resource.WithPollInterval and
// resource.WithWaitStrategy. A zero or negative timeout or interval and other
// controls fail with resource.ErrInvalidOption before any request.
func WithCreateVolumeBackupWaitOptions(options ...resource.WaitOption) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateWaitOptions(options...)
}

func PrepareCreateVolumeBackupOptions(ctx context.Context, options ...CreateVolumeBackupOption) (CreateVolumeBackupOpts, error) {
	return cloudsnapshot.PrepareBackupCreate(ctx, options...)
}
