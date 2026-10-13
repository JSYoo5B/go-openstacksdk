package services

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordResolveOpts contains only explicit Go transport/location conveniences.
// The named pinned identity.find_service has no **query; raw fetch kwargs are
// deliberately absent from this helper used by cloud.get_service.
type RecordResolveOpts struct {
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type RecordResolveOption func(*RecordResolveOpts) error

func copyRecordResolveOpts(v RecordResolveOpts) RecordResolveOpts {
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
func WithRecordResolveOpts(v RecordResolveOpts) RecordResolveOption {
	owned := copyRecordResolveOpts(v)
	return func(c *RecordResolveOpts) error { *c = copyRecordResolveOpts(owned); return nil }
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

// ResolveRecord supplies the closed GET-first Service finder needed by the
// Cloud. Only clean member400/403/404 fall through to a name-hinted all-page
// ID-or-name search. Missing defaults nil and duplicate matches are errors.
func (a *API) ResolveRecord(ctx context.Context, nameOrID string, options ...RecordResolveOption) (*ServiceRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("ResolveRecord", "services", cloudread.ContextError(ctx, err))
	}
	engine, err := a.recordListEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy := RecordResolveOpts{}
	for _, apply := range slices.Clone(options) {
		if err = engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(recordInvalid("nil service resolve option"))
		}
		next := copyRecordResolveOpts(policy)
		if err = errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyRecordResolveOpts(next)
	}
	if policy.Location != nil {
		_, err = policy.Location.ForResource(nil, policy.Location.Zone)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	value, err := engine.Find(ctx, nameOrID, neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion})
	if err != nil {
		return nil, wrap(errors.Join(err, engine.Check(ctx)))
	}
	row, err := recordListRecord(value, RecordListOpts{Location: policy.Location})
	return row, wrap(errors.Join(err, engine.Check(ctx)))
}
