package cloudbackup

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

type ExportResponse = cloudsnapshot.BackupExportResponse
type ExportResult = cloudsnapshot.BackupExportResult

func Export(ctx context.Context, client *gophercloud.ServiceClient, id string) (*ExportResult, error) {
	return cloudsnapshot.ExportBackup(ctx, client, id)
}
