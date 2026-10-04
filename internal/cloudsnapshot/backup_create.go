package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

// CreateBackup forwards the fixed cloud body and preserves request-seeded
// logical state separately from each physical creation and wait response.
func CreateBackup(ctx context.Context, client *gophercloud.ServiceClient, volumeID string, options ...BackupCreateOption) (*BackupCreateResult, error) {
	p, err := captureWithSchema(ctx, client, backupReadSchema())
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "CreateVolumeBackup", err)
	}
	policy, err := prepare(ctx, options, cloneBackupCreate, func() error { return p.source.Guard(ctx) })
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "CreateVolumeBackup", err)
	}
	body, state, err := compileBackupCreateBody(volumeID, policy)
	if err == nil {
		err = p.ownLocation(ctx, policy.Location)
	}
	if err == nil {
		_, err = state.view(p.location) // Source constructor eagerly consumes all Backup descriptors.
	}
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "CreateVolumeBackup", err)
	}
	target, err := p.target("backups")
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "CreateVolumeBackup", err)
	}
	result := &BackupCreateResult{}
	wire, err := p.mutationExchange(ctx, http.MethodPost, target, body)
	result.Created, result.LastAccepted = mutationProof(wire), mutationProof(wire)
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "CreateVolumeBackup", err)
	}
	view, actual, err := p.mergeBackupMutation(ctx, &state, wire)
	result.CreatedBackup, result.BackupID = actual.Clone(), state.id()
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "CreateVolumeBackup", err)
	}
	result.CreatedValue = bytes.Clone(view)
	wait := policy.Wait == nil || *policy.Wait
	if wait {
		view, actual, err = p.waitBackupCreate(ctx, &state, view, actual, policy.WaitPolicy, result)
		if err != nil {
			return result, wrapRead(ctx, backupReadSchema(), "CreateVolumeBackup", err)
		}
	}
	if err := p.source.Guard(ctx); err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "CreateVolumeBackup", err)
	}
	result.Value, result.Backup = bytes.Clone(view), actual.Clone()
	return result, nil
}

func (p *reader) waitBackupCreate(ctx context.Context, state *backupMutationState, view json.RawMessage, actual *resource.RawResource, policy MutationWaitOptions, result *BackupCreateResult) (json.RawMessage, *resource.RawResource, error) {
	status, err := state.status(true)
	if err != nil {
		return nil, nil, err
	}
	if strings.EqualFold(status, "available") {
		if err := p.source.Guard(ctx); err != nil {
			return nil, nil, err
		}
		result.Ready, result.ReadyBackup = bytes.Clone(view), actual.Clone()
		return view, actual, nil
	}
	wait := newMutationWait(policy)
	for {
		if err := wait.boundary(ctx, p); err != nil {
			return nil, nil, err
		}
		id, err := state.routeID()
		if err != nil {
			return nil, nil, err
		}
		target, err := p.target("backups", url.PathEscape(id))
		if err != nil {
			return nil, nil, err
		}
		wire, err := p.mutationExchange(ctx, http.MethodGet, target, nil)
		if wire != nil {
			result.LastAccepted = mutationProof(wire)
		}
		if err != nil {
			return nil, nil, err
		}
		view, actual, err = p.mergeBackupMutation(ctx, state, wire)
		result.BackupID = state.id()
		if err != nil {
			return nil, actual, err
		}
		status, err = state.status(true)
		if err != nil {
			return nil, actual, err
		}
		if strings.EqualFold(status, "available") {
			if err := p.source.Guard(ctx); err != nil {
				return nil, actual, err
			}
			result.Ready, result.ReadyBackup = bytes.Clone(view), actual.Clone()
			return view, actual, nil
		}
		if strings.EqualFold(status, "error") {
			return nil, actual, &resource.FailedStateError{Resource: "volume backup", ID: id, Status: status}
		}
		if err := wait.sleep(ctx, p); err != nil {
			return nil, actual, err
		}
	}
}
