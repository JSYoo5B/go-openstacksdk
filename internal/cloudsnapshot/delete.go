package cloudsnapshot

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/gophercloud/gophercloud/v2"
)

// Delete resolves exactly one snapshot through the same captured source as
// its bodyless DELETE. Waiting is opt-in; a rejected DELETE404 remains error.
func Delete(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...DeleteOption) (*DeleteResult, error) {
	if nameOrID == "" {
		_, err := PrepareDelete(ctx, options...)
		if err == nil {
			err = cloudread.Context(ctx)
		}
		if err != nil {
			return nil, wrap(ctx, "DeleteVolumeSnapshot", err)
		}
		return &DeleteResult{Deleted: clonePointer(new(bool))}, nil
	}
	p, err := capture(ctx, client)
	if err != nil {
		return nil, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	policy, err := prepare(ctx, options, cloneDeleteOptions, func() error { return p.source.Guard(ctx) })
	if err == nil {
		err = p.ownLocation(ctx, policy.Location)
	}
	if err != nil {
		return nil, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	result := &DeleteResult{Resolved: &Result{}}
	value, err := p.collection(result.Resolved).FindIdentity(ctx, nameOrID)
	if err != nil {
		if ctx.Err() != nil && p.memberFailure != nil {
			err = errors.Join(p.memberFailure, err)
		}
		return result, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	if err := p.source.Guard(ctx); err != nil {
		return result, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	if value == nil {
		result.Deleted = clonePointer(new(bool))
		return result, nil
	}
	result.Resolved.Value, result.Resolved.Snapshot = bytes.Clone(value.view), value.resource.Clone()
	if result.Resolved.Observed != nil {
		result.Resolved.RequestedID, result.Resolved.SeededID = nameOrID, value.seeded
	}
	// The reader view preserves selected aliases and member-ID seeding. These
	// already-converted values are semantic prior state, not raw wire proof.
	var state mutationState
	if err := state.overlay(value.view); err != nil {
		return result, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	result.SnapshotID = state.id()
	id, err := state.routeID()
	if err != nil {
		return result, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	target, err := p.target("snapshots", url.PathEscape(id))
	if err != nil {
		return result, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	wire, err := p.mutationExchange(ctx, http.MethodDelete, target, nil)
	result.Applied, result.LastAccepted = mutationProof(wire), mutationProof(wire)
	if err != nil {
		return result, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	if policy.Wait != nil && *policy.Wait {
		if err := p.waitDelete(ctx, &state, policy.WaitPolicy, result); err != nil {
			return result, wrap(ctx, "DeleteVolumeSnapshot", err)
		}
	}
	if err := p.source.Guard(ctx); err != nil {
		return result, wrap(ctx, "DeleteVolumeSnapshot", err)
	}
	complete := true
	result.Deleted = clonePointer(&complete)
	return result, nil
}

func (p *reader) waitDelete(ctx context.Context, state *mutationState, policy MutationWaitOptions, result *DeleteResult) error {
	wait := newMutationWait(policy)
	for {
		if err := wait.boundary(ctx, p); err != nil {
			return err
		}
		id, err := state.routeID()
		if err != nil {
			return err
		}
		target, err := p.target("snapshots", url.PathEscape(id))
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
		view, actual, err := p.mergeMutation(ctx, state, wire)
		result.SnapshotID = state.id()
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
			result.Ready, result.ReadySnapshot = bytes.Clone(view), actual.Clone()
			return nil
		}
		if err := wait.sleep(ctx, p); err != nil {
			return err
		}
	}
}
