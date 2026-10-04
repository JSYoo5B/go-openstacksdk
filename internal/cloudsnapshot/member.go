package cloudsnapshot

import (
	"bytes"
	"context"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
)

func (p *reader) member(ctx context.Context, id string, result *Result) (value *entry, err error) {
	defer func() { p.memberFailure = err }()
	if err := p.source.Guard(ctx); err != nil {
		return nil, err
	}
	target, err := p.target(p.schema.route, url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	wire, err := p.source.Get(ctx, target, sourceCodes()...)
	result.Observed = observed(wire)
	if err != nil {
		return nil, err
	}
	raw, err := memberObjectFor(wire, p.schema)
	if err != nil {
		return nil, err
	}
	return p.materialize(ctx, raw, &id, false, wire)
}

// ByID performs one logical bodyless member GET; no list fallback is possible.
func ByID(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...ReadOption) (*Result, error) {
	p, err := capture(ctx, client)
	if err != nil {
		return nil, wrap(ctx, "GetVolumeSnapshotByID", err)
	}
	policy, err := prepare(ctx, options, cloneRead, func() error { return p.source.Guard(ctx) })
	if err == nil {
		err = ValidateID(id)
	}
	if err == nil {
		err = p.ownLocation(ctx, policy.Location)
	}
	if err != nil {
		return nil, wrap(ctx, "GetVolumeSnapshotByID", err)
	}
	result := &Result{}
	value, err := p.member(ctx, id, result)
	if err == nil {
		err = p.source.Guard(ctx)
	}
	if err != nil {
		return result, wrap(ctx, "GetVolumeSnapshotByID", err)
	}
	result.Value, result.Snapshot = bytes.Clone(value.view), value.resource
	result.RequestedID, result.SeededID = id, value.seeded
	return result, nil
}
