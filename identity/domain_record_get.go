package identity

import (
	"context"
	"errors"
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DomainGetOpts preserves domain_id/name_or_id presence. DeprecatedFilters is
// only a warning flag: ignored filters are never marshaled, parsed or read.
type DomainGetOpts struct {
	DomainID          *string
	NameOrID          *string
	DeprecatedFilters bool
	Headers           map[string]string
	Microversion      *string
	Location          *resource.CloudLocation
}

// DomainGetOption is one option of GetDomain: WithDomainGetOpts,
// WithDomainGetID, WithDomainGetNameOrID, WithDomainGetFilters,
// WithDomainGetLocation, request.Headers (replaces the headers),
// request.Microversion or request.Custom over *DomainGetOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type DomainGetOption = request.GetCall

var domainGetCalls = calls.Spec[DomainGetOpts]{
	Headers:      calls.Value(func(c *DomainGetOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *DomainGetOpts) **string { return &c.Microversion }),
}

func copyDomainGetOpts(v DomainGetOpts) DomainGetOpts {
	if v.DomainID != nil {
		x := *v.DomainID
		v.DomainID = &x
	}
	if v.NameOrID != nil {
		x := *v.NameOrID
		v.NameOrID = &x
	}
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
func WithDomainGetOpts(v DomainGetOpts) DomainGetOption {
	owned := copyDomainGetOpts(v)
	return request.Custom(func(c *DomainGetOpts) error { *c = copyDomainGetOpts(owned); return nil })
}
func WithDomainGetID(v string) DomainGetOption {
	return request.Custom(func(c *DomainGetOpts) error { x := v; c.DomainID = &x; return nil })
}
func WithDomainGetNameOrID(v string) DomainGetOption {
	return request.Custom(func(c *DomainGetOpts) error { x := v; c.NameOrID = &x; return nil })
}
func WithDomainGetFilters(v any) DomainGetOption {
	present := v != nil
	return request.Custom(func(c *DomainGetOpts) error { c.DeprecatedFilters = present; return nil })
}
func WithDomainGetLocation(v resource.CloudLocation) DomainGetOption {
	owned := v.Clone()
	return request.Custom(func(c *DomainGetOpts) error { x := owned.Clone(); c.Location = &x; return nil })
}

// DomainGetResult carries the ignored-filter warning independently of actual
// Resource/evidence. Record nil with nil error means closed-finder absence.
type DomainGetResult struct {
	Record            *domains.DomainRecord
	DeprecatedFilters bool
}

// GetDomain mirrors the Source presence branch: nonnil domain_id uses direct
// get; nil uses a closed GET-first ignore-missing finder. Both omitted is the
// actual pinned InvalidRequest preflight, despite its misleading None comment.
func (c *DomainReadCloud) GetDomain(ctx context.Context, options ...DomainGetOption) (*DomainGetResult, error) {
	wrap := func(err error) error { return request.Wrap("GetDomain", "domains", cloudread.ContextError(ctx, err)) }
	owned, api, _, guard, err := c.prepare(ctx, nil)
	if err != nil {
		return nil, wrap(err)
	}
	policy := DomainGetOpts{}
	for _, apply := range calls.Translate(options, domainGetCalls) {
		if err := guard(); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(domainReadInvalid("nil domain get option"))
		}
		next := copyDomainGetOpts(policy)
		if err := errors.Join(apply(&next), guard()); err != nil {
			return nil, wrap(err)
		}
		policy = copyDomainGetOpts(next)
	}
	result := &DomainGetResult{DeprecatedFilters: policy.DeprecatedFilters}
	if policy.DomainID == nil && policy.NameOrID == nil {
		return result, wrap(domainReadInvalid("request requires an ID but none was found"))
	}
	opts := []domains.RecordGetOption{domains.WithRecordGetOpts(domains.RecordGetOpts{Headers: policy.Headers, Microversion: policy.Microversion, Location: policy.Location})}
	if policy.DomainID != nil {
		result.Record, err = api.GetRecord(owned, *policy.DomainID, opts...)
	} else {
		result.Record, err = api.ResolveRecord(owned, *policy.NameOrID, opts...)
	}
	return result, wrap(errors.Join(err, guard()))
}
