package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ClosedLookupOpts models only domain_id for the named Cloud getter's fixed
// find call, plus explicit Go transport/location conveniences. It is not the
// general Source find_user(**query) API.
type ClosedLookupOpts struct {
	DomainID     json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type ClosedLookupOption func(*ClosedLookupOpts) error

func copyClosedLookupOpts(v ClosedLookupOpts) ClosedLookupOpts {
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
func WithClosedLookupOpts(v ClosedLookupOpts) ClosedLookupOption {
	owned := copyClosedLookupOpts(v)
	return func(c *ClosedLookupOpts) error { *c = copyClosedLookupOpts(owned); return nil }
}
func WithClosedLookupHeaders(v map[string]string) ClosedLookupOption {
	owned := maps.Clone(v)
	return func(c *ClosedLookupOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithClosedLookupMicroversion(v string) ClosedLookupOption {
	return func(c *ClosedLookupOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithClosedLookupLocation(v resource.CloudLocation) ClosedLookupOption {
	owned := v.Clone()
	return func(c *ClosedLookupOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// ResolveRecord supplies only the fixed GET-first User finder needed by the
// Cloud. Only clean member400/403/404 fall through to a name-hinted all-page
// ID-or-name search. Missing defaults nil and duplicate matches are errors.
func (a *API) ResolveRecord(ctx context.Context, nameOrID string, options ...ClosedLookupOption) (*UserRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("ResolveRecord", "users", cloudread.ContextError(ctx, err))
	}
	engine, err := a.closedLookupEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy := ClosedLookupOpts{}
	for _, apply := range slices.Clone(options) {
		if err = engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(recordInvalid("nil users resolve option"))
		}
		next := copyClosedLookupOpts(policy)
		if err = errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyClosedLookupOpts(next)
	}
	if policy.Location != nil {
		_, err = policy.Location.ForResource(nil, policy.Location.Zone)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
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
	row, err := recordListRecord(value, RecordListOpts{Location: policy.Location})
	return row, wrap(errors.Join(err, engine.Check(ctx)))
}

// WithClosedLookupDomainID accepts the ordinary optional string domain.
func WithClosedLookupDomainID(v string) ClosedLookupOption {
	raw, _ := json.Marshal(v)
	return WithClosedLookupDomainIDJSON(raw)
}

// WithClosedLookupDomainIDJSON preserves Source dynamic JSON domain values.
// Nil means Python None and is still forwarded to find's constructor/query.
func WithClosedLookupDomainIDJSON(raw json.RawMessage) ClosedLookupOption {
	owned := bytes.Clone(raw)
	return func(c *ClosedLookupOpts) error { c.DomainID = bytes.Clone(owned); return nil }
}
func (a *API) closedLookupEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("configured users API required")
	}
	selected := a.client
	profile := recordProfile
	profile.NameQueryHint = true
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return recordInvalid("selected users client changed")
		}
		return nil
	})
}
