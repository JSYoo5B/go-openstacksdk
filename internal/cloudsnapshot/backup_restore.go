package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

type BackupRestoreResult struct {
	BackupID json.RawMessage
	Applied  *MutationPage
	Backup   *resource.RawResource
	Value    json.RawMessage
}

// RestoreBackup merges the restore reply into owned cached Backup fields.
// Physical Backup/Applied evidence never contains invented request fields.
func RestoreBackup(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...BackupRestoreOption) (*BackupRestoreResult, error) {
	const operation = "RestoreVolumeBackup"
	p, err := captureWithSchema(ctx, client, backupReadSchema())
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	policy, err := prepare(ctx, options, cloneBackupRestore, func() error { return p.source.Guard(ctx) })
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	body, state, err := compileBackupRestore(id, policy)
	if err == nil {
		err = p.ownLocation(ctx, policy.Location)
	}
	if err == nil {
		_, err = state.view(p.location)
	}
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	target, err := p.target("backups", url.PathEscape(id), "restore")
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	result := &BackupRestoreResult{BackupID: state.id()}
	wire, err := p.backupPost(ctx, target, body, false)
	result.Applied = mutationProof(wire)
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	view, actual, err := p.mergeBackupRestore(ctx, &state, wire)
	result.BackupID, result.Backup = state.id(), actual.Clone()
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), operation, err)
	}
	result.Value = bytes.Clone(view)
	return result, nil
}

func (p *reader) mergeBackupRestore(ctx context.Context, state *backupMutationState, wire *rest.Response) (json.RawMessage, *resource.RawResource, error) {
	// Source response selection prefers restore, then Backup's own key, then
	// the flat object. A present null or wrong shape must not fall through.
	key := "restore"
	var envelope map[string]json.RawMessage
	if json.Unmarshal(wire.Body, &envelope) == nil {
		if _, present := envelope[key]; !present {
			if _, present := envelope["backup"]; present {
				key = "backup"
			}
		}
	}
	raw, actual, err := mutationObjectFor(wire, key)
	if err == nil && raw != nil {
		err = state.overlay(raw)
	}
	var view json.RawMessage
	if err == nil {
		view, err = state.view(p.location)
	}
	if err := errors.Join(err, p.source.Guard(ctx)); err != nil {
		return nil, actual, wire.Fail(err)
	}
	return view, actual, nil
}
