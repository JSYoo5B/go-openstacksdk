package blockstorage

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudbackup"
	"github.com/gophercloud/gophercloud/v2"
)

type ExportVolumeBackupRecordRequest struct{ BackupID string }

// ExportVolumeBackupRecordResult retains an arbitrary owned JSON Value and
// independently owned physical Exported evidence, including on parse failure.
type ExportVolumeBackupRecordResult = cloudbackup.ExportRecordResult

// ExportVolumeBackupRecord validates one complete UTF-8 JSON value of any shape.
// It preserves JSON literals and never retries an accepted response parse error.
func ExportVolumeBackupRecord(ctx context.Context, client *gophercloud.ServiceClient, input ExportVolumeBackupRecordRequest) (*ExportVolumeBackupRecordResult, error) {
	return cloudbackup.ExportRecord(ctx, client, input.BackupID)
}
