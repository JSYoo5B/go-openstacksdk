package blockstorage

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type CreateVolumeBackupRequest struct{ VolumeID string }
type CreateVolumeBackupOpts = cloudsnapshot.BackupCreateOptions
type CreateVolumeBackupOption = cloudsnapshot.BackupCreateOption
type BackupMutationWaitOpts = cloudsnapshot.MutationWaitOptions

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
func WithCreateVolumeBackupWaitPolicy(value BackupMutationWaitOpts) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateWaitPolicy(value)
}
func WithCreateVolumeBackupLocation(value resource.CloudLocation) CreateVolumeBackupOption {
	return cloudsnapshot.WithBackupCreateLocation(value)
}

func PrepareCreateVolumeBackupOptions(ctx context.Context, options ...CreateVolumeBackupOption) (CreateVolumeBackupOpts, error) {
	return cloudsnapshot.PrepareBackupCreate(ctx, options...)
}
