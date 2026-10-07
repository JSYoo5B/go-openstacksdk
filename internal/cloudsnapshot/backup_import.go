package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type BackupImportResult struct {
	BackupID     json.RawMessage
	Microversion string
	Discovery    []*MutationPage
	Applied      *MutationPage
	Backup       *resource.RawResource
	Value        json.RawMessage
}

// The record is an opaque locator, not bytes to base64-encode or a URL to fetch.
func ValidateBackupImportInput(service, record string) error {
	if !utf8.ValidString(service) || !utf8.ValidString(record) {
		return invalid("backup service and record must be UTF-8 strings")
	}
	return nil
}

// ImportBackup repairs the pinned import_record route/argument overwrite.
// The returned logical Backup is disconnected, so no call location is read.
func ImportBackup(ctx context.Context, client *gophercloud.ServiceClient, service, record string) (*BackupImportResult, error) {
	const operation = "ImportVolumeBackup"
	p, err := captureWithSchema(ctx, client, backupReadSchema())
	if err == nil {
		err = ValidateBackupImportInput(service, record)
	}
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	target, err := p.target("backups", "import_record")
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	result := &BackupImportResult{BackupID: json.RawMessage("null")}
	result.Microversion, result.Discovery, err = p.backupImportMicroversion(ctx)
	if err == nil {
		err = p.source.WithPolicy(ctx, &result.Microversion, nil)
	}
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	body, _ := json.Marshal(map[string]any{"backup-record": map[string]string{"backup_service": service, "backup_url": record}})
	wire, err := p.backupPostPolicy(ctx, target, body, &result.Microversion)
	result.Applied = mutationProof(wire)
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	raw, actual, err := mutationObjectFor(wire, "backup")
	result.Backup = actual.Clone()
	var state backupMutationState
	if err == nil && raw != nil {
		err = state.overlay(raw)
	}
	result.BackupID = state.id()
	var view json.RawMessage
	if err == nil {
		view, err = normalizeBackupDisconnected(state.object())
	}
	if err = errors.Join(err, p.source.Guard(ctx)); err != nil {
		return result, wrapRead(ctx, backupReadSchema(), operation, wire.Fail(err))
	}
	result.Value = bytes.Clone(view)
	return result, nil
}
