package domains

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordGetOpts owns explicit Go transport/location conveniences. The pinned
// Proxy.get_domain has no arbitrary fetch/query controls, and none are added.
type RecordGetOpts struct {
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type RecordGetOption func(*RecordGetOpts) error

func copyRecordGetOpts(v RecordGetOpts) RecordGetOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	return v
}
func WithRecordGetOpts(v RecordGetOpts) RecordGetOption {
	owned := copyRecordGetOpts(v)
	return func(c *RecordGetOpts) error { *c = copyRecordGetOpts(owned); return nil }
}
func WithRecordGetHeaders(v map[string]string) RecordGetOption {
	owned := maps.Clone(v)
	return func(c *RecordGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordGetMicroversion(v string) RecordGetOption {
	return func(c *RecordGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordGetLocation(v resource.CloudLocation) RecordGetOption {
	owned := v.Clone()
	return func(c *RecordGetOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}
func (a *API) prepareRecordGet(ctx context.Context, options []RecordGetOption, resolve bool) (*neutronfind.Engine, RecordGetOpts, error) {
	engine, err := a.recordEngine(ctx)
	if err != nil {
		return nil, RecordGetOpts{}, err
	}
	if resolve {
		selected := a.client
		profile := domainRecordProfile
		profile.NameQueryHint = true
		engine, err = neutronfind.New(ctx, selected, profile, func(context.Context) error {
			if a.client != selected {
				return recordInvalid("selected domain resolver client changed")
			}
			return nil
		})
		if err != nil {
			return nil, RecordGetOpts{}, err
		}
	}
	policy, err := applyDomainOptions(ctx, engine, RecordGetOpts{}, copyRecordGetOpts, slices.Clone(options))
	if err == nil {
		err = locationPreflight(policy.Location)
	}
	return engine, policy, errors.Join(err, engine.Check(ctx))
}
func projectRecordGet(ctx context.Context, engine *neutronfind.Engine, value *neutronfind.Record, policy RecordGetOpts, err error) (*DomainRecord, error) {
	row := domainRecord(value)
	if err == nil {
		err = errors.Join(recordLocation(row, policy.Location), engine.Check(ctx))
		if err != nil && row != nil {
			err = &resource.ResponseError{Body: bytes.Clone(row.Envelope), Header: row.Header.Clone(), StatusCode: row.StatusCode, Cause: err}
		}
	}
	if err != nil && row != nil {
		row.Resource = nil
	}
	return row, err
}

// GetRecord makes one literal member GET with nullable Source descriptors,
// seeded request ID and original response channels. It never falls back.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordGetOption) (*DomainRecord, error) {
	const op = "GetRecord"
	engine, policy, err := a.prepareRecordGet(ctx, options, false)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	value, err := engine.Get(ctx, id, neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion})
	row, err := projectRecordGet(ctx, engine, value, policy, err)
	return row, recordWrap(ctx, op, err)
}

// ResolveRecord is the closed ignore-missing Domain resolver used by cloud
// get_domain. It does not expose named Proxy.find_domain's arbitrary controls.
func (a *API) ResolveRecord(ctx context.Context, nameOrID string, options ...RecordGetOption) (*DomainRecord, error) {
	const op = "ResolveRecord"
	engine, policy, err := a.prepareRecordGet(ctx, options, true)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	value, err := engine.Find(ctx, nameOrID, neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion})
	row, err := projectRecordGet(ctx, engine, value, policy, err)
	return row, recordWrap(ctx, op, cloudread.ContextError(ctx, err))
}
