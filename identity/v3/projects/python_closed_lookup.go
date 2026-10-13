package projects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"maps"
	"slices"
)

// RecordResolveOpts is the closed domain-scoped finder used by cloud get.
// DomainID retains literal JSON; omission means the Source domain_id=None.
// No generic fetch kwargs, URI/session replacement or cache controls exist.
type RecordResolveOpts struct {
	DomainID     json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type RecordResolveOption func(*RecordResolveOpts) error

func copyResolveOptions(v RecordResolveOpts) RecordResolveOpts {
	v.DomainID = bytes.Clone(v.DomainID)
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
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
func WithRecordResolveOpts(v RecordResolveOpts) RecordResolveOption {
	owned := copyResolveOptions(v)
	return func(c *RecordResolveOpts) error { *c = copyResolveOptions(owned); return nil }
}
func WithRecordResolveDomainID(v string) RecordResolveOption {
	raw, _ := json.Marshal(v)
	return WithRecordResolveRawDomainID(raw)
}
func WithRecordResolveRawDomainID(v json.RawMessage) RecordResolveOption {
	owned := bytes.Clone(v)
	return func(c *RecordResolveOpts) error { c.DomainID = bytes.Clone(owned); return nil }
}
func WithRecordResolveHeaders(v map[string]string) RecordResolveOption {
	owned := maps.Clone(v)
	return func(c *RecordResolveOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordResolveMicroversion(v string) RecordResolveOption {
	return func(c *RecordResolveOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordResolveLocation(v resource.CloudLocation) RecordResolveOption {
	owned := v.Clone()
	return func(c *RecordResolveOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// ResolveRecord closes Source find to identity/domain_id and explicit Go
// transport/location conveniences. It never claims full named Proxy parity.
func (a *API) ResolveRecord(ctx context.Context, nameOrID string, options ...RecordResolveOption) (*ProjectRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("ResolveRecord", "projects", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("configured projects API required"))
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, recordFindProfile, func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected projects client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	policy := copyResolveOptions(RecordResolveOpts{})
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil closed resolve option"))
		}
		next := copyResolveOptions(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyResolveOptions(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return nil, wrap(errors.Join(err, engine.Check(ctx)))
		}
	}
	domain := bytes.Clone(policy.DomainID)
	if domain == nil {
		domain = json.RawMessage("null")
	}
	value, err := engine.Find(ctx, nameOrID, neutronfind.Options{Query: map[string]json.RawMessage{"domain_id": domain}, Order: []string{"domain_id"}, Headers: policy.Headers, Microversion: policy.Microversion})
	if err != nil {
		return nil, wrap(errors.Join(err, engine.Check(ctx)))
	}
	row := projectListRecord(value)
	if row != nil && policy.Location != nil {
		raw, cause := policy.Location.ForResource(row.Resource.Body["project_id"], row.Resource.Body["availability_zone"])
		err = errors.Join(err, cause)
		if cause == nil {
			row.Resource.Body["location"] = bytes.Clone(raw)
		}
	}
	return row, wrap(errors.Join(err, engine.Check(ctx)))
}
