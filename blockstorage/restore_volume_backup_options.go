package blockstorage

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
)

type RestoreVolumeBackupRequest struct{ BackupID string }
type RestoreVolumeBackupOpts = cloudsnapshot.BackupRestoreOptions
type RestoreVolumeBackupOption = cloudsnapshot.BackupRestoreOption

func WithRestoreVolumeBackupOptions(value RestoreVolumeBackupOpts) RestoreVolumeBackupOption {
	return cloudsnapshot.WithBackupRestoreOptions(value)
}
func WithRestoreVolumeBackupVolumeID(value string) RestoreVolumeBackupOption {
	return cloudsnapshot.WithBackupRestoreVolumeID(value)
}
func WithRestoreVolumeBackupName(value string) RestoreVolumeBackupOption {
	return cloudsnapshot.WithBackupRestoreName(value)
}
func WithRestoreVolumeBackupSeed(value json.RawMessage) RestoreVolumeBackupOption {
	return cloudsnapshot.WithBackupRestoreSeed(value)
}

func PrepareRestoreVolumeBackupOptions(ctx context.Context, options ...RestoreVolumeBackupOption) (RestoreVolumeBackupOpts, error) {
	return cloudsnapshot.PrepareBackupRestore(ctx, options...)
}
