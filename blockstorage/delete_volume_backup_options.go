package blockstorage

import (
	"context"
	"gophercloudsdk/internal/cloudsnapshot"
	"gophercloudsdk/resource"
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
func WithDeleteVolumeBackupWaitPolicy(value BackupMutationWaitOpts) DeleteVolumeBackupOption {
	return cloudsnapshot.WithBackupDeleteWaitPolicy(value)
}
func WithDeleteVolumeBackupLocation(value resource.CloudLocation) DeleteVolumeBackupOption {
	return cloudsnapshot.WithBackupDeleteLocation(value)
}

func PrepareDeleteVolumeBackupOptions(ctx context.Context, options ...DeleteVolumeBackupOption) (DeleteVolumeBackupOpts, error) {
	return cloudsnapshot.PrepareBackupDelete(ctx, options...)
}
