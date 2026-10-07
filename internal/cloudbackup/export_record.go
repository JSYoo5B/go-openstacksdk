package cloudbackup

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

type ExportRecordResult = cloudsnapshot.BackupExportRecordResult

func ExportRecord(ctx context.Context, client *gophercloud.ServiceClient, id string) (*ExportRecordResult, error) {
	return cloudsnapshot.ExportBackupRecord(ctx, client, id)
}
