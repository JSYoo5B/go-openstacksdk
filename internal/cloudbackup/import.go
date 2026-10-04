package cloudbackup

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudsnapshot"
)

type ImportResult = cloudsnapshot.BackupImportResult

func Import(ctx context.Context, client *gophercloud.ServiceClient, service, record string) (*ImportResult, error) {
	return cloudsnapshot.ImportBackup(ctx, client, service, record)
}
