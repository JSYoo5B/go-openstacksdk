package blockstorage

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudbackup"
	"github.com/gophercloud/gophercloud/v2"
)

type ExportVolumeBackupRequest struct{ BackupID string }

// VolumeBackupExportResponse owns opaque accepted body/header/status evidence.
// The body is not decoded as a BackupRecord, JSON, UTF-8, or base64 data.
type VolumeBackupExportResponse = cloudbackup.ExportResponse

// ExportVolumeBackupResult retains an admitted response on partial IO failure.
// A nonnil Exported response may accompany an error; success requires nil error.
type ExportVolumeBackupResult = cloudbackup.ExportResult

// ExportVolumeBackup sends a bodyless export GET for one literal backup ID.
// It preserves the selected microversion and performs no lookup or wait.
func ExportVolumeBackup(ctx context.Context, cinder *gophercloud.ServiceClient, input ExportVolumeBackupRequest) (*ExportVolumeBackupResult, error) {
	return cloudbackup.Export(ctx, cinder, input.BackupID)
}
