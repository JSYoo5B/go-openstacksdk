package users

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/keystonecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordFindOpts owns Source kwargs. Omitted IgnoreMissing defaults to true.
// Query preserves JSON values, including arrays and null; unknown kwargs go to
// the member GET but are ignored by the list, following Source Resource.find.
// requires_id, base_path, skip_cache, error_message and resource_response_key
// control the GET stage; raw headers stays a GET query value and supplies HTTP
// headers only if the lookup falls back to LIST.
type RecordFindOpts struct {
	ListBasePath  *string
	IgnoreMissing *bool
	Microversion  *string
	queryOrder    []string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Location      *resource.CloudLocation
}

// RecordFindOption is one option of FindRecord: WithRecordFindOpts, WithRecordFindListBasePath, WithRecordFindLocation,
// request.Custom over *RecordFindOpts, request.Query (a Source keyword
// argument; the value is sent as JSON), request.Headers (replaces the
// headers), request.Microversion or request.IgnoreMissing. Other shared
// calls, including request.Header, fail with resource.ErrInvalidOption
// before HTTP. The fallback list route stays WithRecordFindListBasePath:
// finds do not accept the shared BasePath call.
type RecordFindOption = request.FindCall

var recordFindCalls = keystonecalls.FindSpec(func(c *RecordFindOpts) keystonecalls.FindState {
	return keystonecalls.FindState{Query: &c.Query, Order: &c.queryOrder, Headers: &c.Headers, Microversion: &c.Microversion, IgnoreMissing: &c.IgnoreMissing}
})

func internalFindOpts(value RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{ListBasePath: value.ListBasePath, IgnoreMissing: value.IgnoreMissing, Microversion: value.Microversion, Query: value.Query, Headers: value.Headers, Order: value.queryOrder}
}
func copyFindOpts(value RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(internalFindOpts(value))
	value.ListBasePath = owned.ListBasePath
	value.IgnoreMissing, value.Microversion = owned.IgnoreMissing, owned.Microversion
	value.Query, value.Headers, value.queryOrder = owned.Query, owned.Headers, owned.Order
	if value.Location != nil {
		x := value.Location.Clone()
		value.Location = &x
	}
	return value
}

func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyFindOpts(value)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil })
}

// FindRecord matches the pinned Source GET-first, complete ID-or-name search.
// It suppresses only clean GET 400/403/404. A physical body/Close/source/cancel
// failure never becomes a fallback or missing-resource success.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*UserRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "users", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(recordInvalid("users API is required"))
	}
	engine, err := a.closedLookupEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	guard := engine.Check
	policy := RecordFindOpts{}
	for _, apply := range calls.Translate(options, recordFindCalls) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(recordInvalid("nil find option"))
		}
		next := copyFindOpts(policy)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return nil, wrap(err)
		}
	}
	value, err := engine.FindWithFetchControls(ctx, identity, internalFindOpts(policy))
	if value == nil {
		return nil, wrap(err)
	}
	result, projectErr := recordListRecord(value, RecordListOpts{Location: policy.Location})
	return result, wrap(errors.Join(err, projectErr, engine.Check(ctx)))
}

// WithRecordFindListBasePath overrides only the clean-fallback LIST route.
func WithRecordFindListBasePath(v string) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { x := v; c.ListBasePath = &x; return nil })
}

// WithRecordFindLocation supplies explicit immutable Connection facts.
func WithRecordFindLocation(v resource.CloudLocation) RecordFindOption {
	owned := v.Clone()
	return request.Custom(func(c *RecordFindOpts) error { x := owned.Clone(); c.Location = &x; return nil })
}
