package cloudsnapshot

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// DeleteBackup resolves one exact Backup through one captured Cinder source.
// Force uses the Backup action policy; either mutation404 remains terminal.
func DeleteBackup(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...BackupDeleteOption) (*BackupDeleteResult, error) {
	if nameOrID == "" {
		_, err := PrepareBackupDelete(ctx, options...)
		if err == nil {
			err = cloudread.Context(ctx)
		}
		if err != nil {
			return nil, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
		}
		return &BackupDeleteResult{Deleted: clonePointer(new(bool))}, nil
	}
	p, err := captureWithSchema(ctx, client, backupReadSchema())
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	policy, err := prepare(ctx, options, cloneBackupDelete, func() error { return p.source.Guard(ctx) })
	if err == nil {
		err = p.ownLocation(ctx, policy.Location)
	}
	if err != nil {
		return nil, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	result := &BackupDeleteResult{}
	resolved := &Result{}
	defer func() { result.Resolved = backupResolved(resolved) }()
	value, err := p.collection(resolved).FindIdentity(ctx, nameOrID)
	if err != nil {
		if ctx.Err() != nil && p.memberFailure != nil {
			err = errors.Join(p.memberFailure, err)
		}
		return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	if err := p.source.Guard(ctx); err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	if value == nil {
		result.Deleted = clonePointer(new(bool))
		return result, nil
	}
	resolved.Value, resolved.Snapshot = bytes.Clone(value.view), value.resource.Clone()
	if resolved.Observed != nil {
		resolved.RequestedID, resolved.SeededID = nameOrID, value.seeded
	}
	// The reader view preserves selected aliases and member-ID seeding. These
	// already-converted values are semantic prior state, not raw wire proof.
	var state backupMutationState
	if err := state.overlay(value.view); err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	result.BackupID = state.id()
	id, err := state.routeID()
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	target, err := p.target("backups", url.PathEscape(id))
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	var wire *rest.Response
	if policy.Force != nil && *policy.Force {
		target, err = p.target("backups", url.PathEscape(id), "action")
		if err != nil {
			return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
		}
		wire, err = p.backupForceDelete(ctx, target)
	} else {
		wire, err = p.mutationExchange(ctx, http.MethodDelete, target, nil)
	}
	result.Applied, result.LastAccepted = mutationProof(wire), mutationProof(wire)
	if err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	if policy.Wait != nil && *policy.Wait {
		if err := p.waitBackupDelete(ctx, &state, policy.WaitPolicy, result); err != nil {
			return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
		}
	}
	if err := p.source.Guard(ctx); err != nil {
		return result, wrapRead(ctx, backupReadSchema(), "DeleteVolumeBackup", err)
	}
	complete := true
	result.Deleted = clonePointer(&complete)
	return result, nil
}

func (p *reader) waitBackupDelete(ctx context.Context, state *backupMutationState, policy MutationWaitOptions, result *BackupDeleteResult) error {
	wait := newMutationWait(policy)
	for {
		if err := wait.boundary(ctx, p); err != nil {
			return err
		}
		id, err := state.routeID()
		if err != nil {
			return err
		}
		target, err := p.target("backups", url.PathEscape(id))
		if err != nil {
			return err
		}
		wire, err := p.deletePoll(ctx, target)
		if wire != nil {
			result.LastAccepted = mutationProof(wire)
		}
		if err != nil {
			// Expanded native OkCodes, callback wrappers and body/Close errors
			// cannot establish absence merely by containing a native404.
			if native, clean := err.(gophercloud.ErrUnexpectedResponseCode); clean && native.Actual == http.StatusNotFound {
				result.Absent = &MutationPage{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
				return p.source.Guard(ctx)
			}
			return err
		}
		view, actual, err := p.mergeBackupMutation(ctx, state, wire)
		result.BackupID = state.id()
		if err != nil {
			return err
		}
		status, err := state.status(false)
		if err != nil {
			return err
		}
		if strings.EqualFold(status, "deleted") {
			if err := p.source.Guard(ctx); err != nil {
				return err
			}
			result.Ready, result.ReadyBackup = bytes.Clone(view), actual.Clone()
			return nil
		}
		if err := wait.sleep(ctx, p); err != nil {
			return err
		}
	}
}

func backupResolved(value *Result) *BackupResult {
	return &BackupResult{Value: value.Value, Backup: value.Snapshot, Observed: value.Observed,
		Pages: value.Pages, RequestedID: value.RequestedID, SeededID: value.SeededID}
}
