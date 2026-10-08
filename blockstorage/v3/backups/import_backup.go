package backups

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudbackup"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

type ImportBackupResult = cloudbackup.ImportResult

func (a *API) ImportBackup(ctx context.Context, service, url string) (*ImportBackupResult, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	result, err := cloudbackup.Import(ctx, client, service, url)
	return result, request.Wrap("ImportBackup", "backups", err)
}
