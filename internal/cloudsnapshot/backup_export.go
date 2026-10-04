package cloudsnapshot

import (
	"bytes"
	"context"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
)

// BackupExportResponse is opaque HTTP evidence, including non-JSON bytes.
type BackupExportResponse struct {
	Body       []byte
	Header     http.Header
	StatusCode int
}

// BackupExportResult retains admitted evidence even on accepted body IO failure.
// Exported does not by itself establish successful completion; inspect error.
type BackupExportResult struct {
	BackupID string
	Exported *BackupExportResponse
}

// ValidateBackupExportID requires one literal, unescaped UTF-8 member segment.
func ValidateBackupExportID(id string) error { return validateIDFor(id, "backup") }

// ExportBackup implements the cloud v3 raw-response helper without resolving
// the backup, normalizing a model, selecting location, or waiting.
func ExportBackup(ctx context.Context, client *gophercloud.ServiceClient, id string) (*BackupExportResult, error) {
	p, err := captureWithSchema(ctx, client, backupReadSchema())
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "ExportVolumeBackup", err)
	}
	if err := ValidateBackupExportID(id); err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "ExportVolumeBackup", err)
	}
	target, err := p.target("backups", url.PathEscape(id), "export_record")
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "ExportVolumeBackup", err)
	}
	result := &BackupExportResult{BackupID: id}
	wire, err := p.backupExportExchange(ctx, target)
	if wire != nil {
		result.Exported = &BackupExportResponse{
			Body: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode,
		}
	}
	if err == nil {
		if guard := p.source.Guard(ctx); guard != nil {
			err = wire.Fail(guard)
		}
	}
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "ExportVolumeBackup", err)
	}
	return result, nil
}
