package cloudsnapshot

import (
	"context"
	"encoding/json"
	"net/url"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
)

type BackupResetStatusResult struct {
	BackupID  string
	Applied   *MutationPage
	Completed bool
}

func ValidateBackupResetStatusInput(id, status string) error {
	if err := ValidateBackupExportID(id); err != nil {
		return err
	}
	if !utf8.ValidString(status) {
		return invalid("backup status must be UTF-8")
	}
	return nil
}

// ResetBackupStatus forwards even an empty or unrecognized UTF-8 status.
// The source action is opaque and has no wait, model update or enum default.
func ResetBackupStatus(ctx context.Context, client *gophercloud.ServiceClient, id, status string) (*BackupResetStatusResult, error) {
	const operation = "ResetVolumeBackupStatus"
	p, err := captureWithSchema(ctx, client, backupReadSchema())
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	if err := ValidateBackupResetStatusInput(id, status); err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	target, err := p.target("backups", url.PathEscape(id), "action")
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	body, _ := json.Marshal(map[string]any{"os-reset_status": map[string]string{"status": status}})
	result := &BackupResetStatusResult{BackupID: id}
	wire, err := p.backupPost(ctx, target, body, true)
	result.Applied = mutationProof(wire)
	if err == nil {
		err = p.source.Guard(ctx)
		if err != nil {
			err = wire.Fail(err)
		}
	}
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	result.Completed = true
	return result, nil
}
