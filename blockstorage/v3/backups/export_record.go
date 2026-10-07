package backups

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudbackup"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

type ExportRecordResponse = cloudbackup.ExportResponse
type ExportRecordResult = cloudbackup.ExportResult
type ExportBackupResult = cloudbackup.ExportRecordResult

// ExportRecord returns the v3 deprecated Proxy export's opaque HTTP evidence.
func (a *API) ExportRecord(ctx context.Context, id string) (*ExportRecordResult, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	result, err := cloudbackup.Export(ctx, client, id)
	return result, request.Wrap("ExportRecord", "backups", err)
}

// ExportBackup returns one complete arbitrary JSON value plus physical evidence.
// Export remains the independently typed native BackupRecord operation.
func (a *API) ExportBackup(ctx context.Context, id string) (*ExportBackupResult, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	result, err := cloudbackup.ExportRecord(ctx, client, id)
	return result, request.Wrap("ExportBackup", "backups", err)
}
