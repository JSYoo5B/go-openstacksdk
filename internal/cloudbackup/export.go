package cloudbackup

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudsnapshot"
)

type ExportResponse = cloudsnapshot.BackupExportResponse
type ExportResult = cloudsnapshot.BackupExportResult

func Export(ctx context.Context, client *gophercloud.ServiceClient, id string) (*ExportResult, error) {
	return cloudsnapshot.ExportBackup(ctx, client, id)
}
