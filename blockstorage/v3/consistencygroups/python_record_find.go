package consistencygroups

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// cloneFindPhase shares the original captured guard/provider facts while
// owning the per-phase client headers, version and policy snapshots.
func (p *recordPrepared) cloneFindPhase() *recordPrepared {
	source := *p.source
	source.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
	// Capture already checked that any selected aliases were equivalent.
	// The private exchange policy reconstructs their selected value; removing
	// aliases here permits a GET override without changing the original client.
	for key := range source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(source.Client.MoreHeaders, key)
		}
	}
	phase := &recordPrepared{ctx: p.ctx, source: &source, guard: p.guard}
	return phase
}

func findRecordText(v *ConsistencyGroupRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[key], &text)
	return text
}

// FindRecord first fetches the literal identity. A clean400/403/404 falls back
// to the complete detailed list, or /consistencygroups for Details(false). ConsistencyGroup declares
// no name query: local exact ID/name matching observes all pages and duplicates.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*ConsistencyGroupRecord, error) {
	const operation = "FindRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, operation, err)
	}
	if err = recordID(identity); err != nil {
		return nil, recordWrap(ctx, operation, err)
	}
	v, err := applyRecord(p, RecordFindOpts{}, copyFindOpts, slices.Clone(options))
	if err != nil {
		return nil, recordWrap(ctx, operation, err)
	}
	// Find has one ordinary per-call header profile. Version selection is an
	// owned GET option and is deliberately not a header-based route override.
	for key := range v.Headers {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			return nil, recordWrap(ctx, operation, recordInvalid("find version header %q is SDK owned", key))
		}
	}
	getPhase := p.cloneFindPhase()
	listPhase := p.cloneFindPhase()
	// Empty override is the documented Go inherit-default policy; Source
	// Resource.fetch forwards an explicitly empty microversion string.
	if v.Microversion != nil && *v.Microversion == "" {
		v.Microversion = nil
	}
	if err = getPhase.policy(v.Microversion, v.Headers); err == nil {
		err = listPhase.policy(nil, v.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, operation, err)
	}
	var lastPage *rest.Response
	get := func(_ context.Context, id string) (*ConsistencyGroupRecord, error) {
		return getPhase.getPrepared(id)
	}
	list := func(context.Context, url.Values) iter.Seq2[*ConsistencyGroupRecord, error] {
		policy := RecordListOpts{Details: v.Details, Headers: maps.Clone(v.Headers)}
		query, filters, control, prepareErr := prepareList(policy)
		if prepareErr == nil {
			prepareErr = listPhase.prepareListPolicy(policy)
		}
		if prepareErr != nil {
			return func(yield func(*ConsistencyGroupRecord, error) bool) { yield(nil, prepareErr) }
		}
		return listPhase.listPrepared(policy, query, filters, control, func(response *rest.Response) { lastPage = response })
	}
	collection := resource.NewCollection(resource.Adapter[ConsistencyGroupRecord]{
		Kind: "consistencygroups", IdentityFind: true, ValidateID: recordID, IdentityDirectGet: recordID,
		IdentityResponseID: func(value *ConsistencyGroupRecord) (string, error) {
			if value == nil || value.Resource == nil {
				return "", recordInvalid("consistency group find response resource is missing")
			}
			// A successful member GET returns Source's untyped id/null unchanged.
			// Only an actual string can equal the caller's string during list matching.
			return findRecordText(value, "id"), nil
		},
		Name: func(value *ConsistencyGroupRecord) string { return findRecordText(value, "name") },
		Get:  get, Iterate: list,
	})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(v.IgnoreMissing == nil || *v.IgnoreMissing))
	var ambiguous *resource.AmbiguousError
	if errors.As(err, &ambiguous) && lastPage != nil {
		err = lastPage.Fail(err)
	}
	return value, recordWrap(ctx, operation, errors.Join(err, p.guard(ctx)))
}

// RecordFindOpts keeps find_consistency_group's IgnoreMissing/Details defaults true.
// Microversion is a GET-only inherited Resource.find override; fallback uses
// the original selected version. Headers are an explicit ordinary Go extension.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Details       *bool
	Microversion  *string
	Headers       map[string]string
}
type RecordFindOption func(*RecordFindOpts) error

func copyFindOpts(v RecordFindOpts) RecordFindOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyFindOpts(v)
	return func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindDetails(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Details = &x; return nil }
}
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := maps.Clone(v)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
