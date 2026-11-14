package backups_test

import (
	"context"

	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
)

func clearBackupMetadataBackups(ctx context.Context, cinder *v3.Service, backupID string) (*backups.Backup, error) {
	opts := backups.UpdateOpts{
		Metadata: map[string]string{},
	}
	return cinder.Backups.Update(ctx, backupID, backups.UpdateOpts{},
		backups.WithUpdateOptions(opts),
	)
}
