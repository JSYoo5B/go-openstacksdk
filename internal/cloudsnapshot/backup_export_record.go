package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
)

// BackupExportRecordResult separates a complete JSON value from physical proof.
// Value is nil on error; successful JSON null is the nonnil bytes "null".
type BackupExportRecordResult struct {
	BackupID string
	Value    json.RawMessage
	Exported *BackupExportResponse
}

// ExportBackupRecord maps the v3 Proxy's arbitrary JSON export result. One
// captured source owns the exchange, pure parser, and final policy check.
func ExportBackupRecord(ctx context.Context, client *gophercloud.ServiceClient, id string) (*BackupExportRecordResult, error) {
	const operation = "ExportVolumeBackupRecord"
	p, err := captureWithSchema(ctx, client, backupReadSchema())
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	if err := ValidateBackupExportID(id); err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	target, err := p.target("backups", url.PathEscape(id), "export_record")
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	result := &BackupExportRecordResult{BackupID: id}
	wire, err := p.backupExportExchange(ctx, target)
	if wire != nil {
		result.Exported = &BackupExportResponse{
			Body: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode,
		}
	}
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	value, parseErr := backupExportJSON(wire.Body)
	if err := errors.Join(parseErr, p.source.Guard(ctx)); err != nil {
		return result, wrapRead(ctx, backupReadSchema(), operation, wire.Fail(err))
	}
	result.Value = value
	return result, nil
}

// Keep arbitrary JSON shapes and number/escape/duplicate literals. Unlike a
// Python dict this owned Go representation does not normalize duplicate keys.
func backupExportJSON(body []byte) (json.RawMessage, error) {
	if !utf8.Valid(body) {
		return nil, fmt.Errorf("backup export response must contain UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("backup export response contains multiple JSON values")
	}
	return bytes.Clone(value), nil
}
