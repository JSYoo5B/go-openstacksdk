package cloudbackup

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudsnapshot"
)

type ExportRecordResult = cloudsnapshot.BackupExportRecordResult

func ExportRecord(ctx context.Context, client *gophercloud.ServiceClient, id string) (*ExportRecordResult, error) {
	return cloudsnapshot.ExportBackupRecord(ctx, client, id)
}
