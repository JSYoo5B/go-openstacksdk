package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Create forwards a literal volume ID and owns defaults, name aliases,
// request-seeded response merging, and the source snapshot-specific wait.
func Create(ctx context.Context, client *gophercloud.ServiceClient, volumeID string, options ...CreateOption) (*CreateResult, error) {
	p, err := capture(ctx, client)
	if err != nil {
		return nil, wrap(ctx, "CreateVolumeSnapshot", err)
	}
	policy, err := prepare(ctx, options, cloneCreateOptions, func() error { return p.source.Guard(ctx) })
	if err != nil {
		return nil, wrap(ctx, "CreateVolumeSnapshot", err)
	}
	body, state, err := compileCreateBody(volumeID, policy)
	if err == nil {
		err = p.ownLocation(ctx, policy.Location)
	}
	if err == nil {
		_, err = state.view(p.location) // Source constructor eagerly consumes all descriptors.
	}
	if err != nil {
		return nil, wrap(ctx, "CreateVolumeSnapshot", err)
	}
	target, err := p.target("snapshots")
	if err != nil {
		return nil, wrap(ctx, "CreateVolumeSnapshot", err)
	}
	result := &CreateResult{}
	wire, err := p.mutationExchange(ctx, http.MethodPost, target, body)
	result.Created, result.LastAccepted = mutationProof(wire), mutationProof(wire)
	if err != nil {
		return result, wrap(ctx, "CreateVolumeSnapshot", err)
	}
	view, actual, err := p.mergeMutation(ctx, &state, wire)
	result.CreatedSnapshot, result.SnapshotID = actual.Clone(), state.id()
	if err != nil {
		return result, wrap(ctx, "CreateVolumeSnapshot", err)
	}
	result.CreatedValue = bytes.Clone(view)
	wait := policy.Wait == nil || *policy.Wait
	if wait {
		view, actual, err = p.waitCreate(ctx, &state, view, actual, policy.WaitPolicy, result)
		if err != nil {
			return result, wrap(ctx, "CreateVolumeSnapshot", err)
		}
	}
	if err := p.source.Guard(ctx); err != nil {
		return result, wrap(ctx, "CreateVolumeSnapshot", err)
	}
	result.Value, result.Snapshot = bytes.Clone(view), actual.Clone()
	return result, nil
}

func (p *reader) waitCreate(ctx context.Context, state *mutationState, view json.RawMessage, actual *resource.RawResource, policy MutationWaitOptions, result *CreateResult) (json.RawMessage, *resource.RawResource, error) {
	status, err := state.status(true)
	if err != nil {
		return nil, nil, err
	}
	if strings.EqualFold(status, "available") {
		if err := p.source.Guard(ctx); err != nil {
			return nil, nil, err
		}
		result.Ready, result.ReadySnapshot = bytes.Clone(view), actual.Clone()
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
		target, err := p.target("snapshots", url.PathEscape(id))
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
		view, actual, err = p.mergeMutation(ctx, state, wire)
		result.SnapshotID = state.id()
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
			result.Ready, result.ReadySnapshot = bytes.Clone(view), actual.Clone()
			return view, actual, nil
		}
		if strings.EqualFold(status, "error") {
			return nil, actual, &resource.FailedStateError{Resource: "volume snapshot", ID: id, Status: status}
		}
		if err := wait.sleep(ctx, p); err != nil {
			return nil, actual, err
		}
	}
}
