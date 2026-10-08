package cloudbackup

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

type ImportResult = cloudsnapshot.BackupImportResult

func Import(ctx context.Context, client *gophercloud.ServiceClient, service, record string) (*ImportResult, error) {
	return cloudsnapshot.ImportBackup(ctx, client, service, record)
}
