package ipsecpolicies

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// policyListProfile owns the pinned VPN policy Body/Query data.
// Source list adds no name hint and classifies kwargs exactly.
func policyListProfile() neutronfind.Profile {
	profile := policyRecordProfile
	profile.Body = slices.Clone(policyRecordProfile.Body)
	profile.Headers = slices.Clone(policyRecordProfile.Headers)
	profile.Query = slices.Clone(policyRecordProfile.Query)
	profile.NameQueryHint = false
	profile.SourceFindPolicy, profile.StopFaultRetryCallbacks = true, true
	profile.PythonLoadedEquality = true
	return profile
}

// RecordListOpts owns JSON Source list kwargs and transport conveniences.
// Query retains omission/null/false/zero. Microversion applies to every page.
// Default pagination is true; max_items 0 is unbounded and limit is omitted.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	BasePath     *string
	expression   json.RawMessage
	order        []string
}
type RecordListOption func(*RecordListOpts) error

func recordListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
}
func copyRecordListOpts(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(recordListOptions(v))
	v.Query, v.Headers, v.Microversion, v.BasePath, v.order = owned.Query, owned.Headers, owned.Microversion, owned.BasePath, owned.Order
	v.expression = bytes.Clone(v.expression)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRecordListOpts(v)
	return func(c *RecordListOpts) error { *c = copyRecordListOpts(owned); return nil }
}

// recordListIgnores reports kwargs that Source Resource.list discards before
// value evaluation: neither a declared query name/wire nor a Body attribute.
func recordListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return false
	}
	switch key {
	case "cls", "paginated", "max_items", "base_path", "resource_type", "session", "microversion", "headers", "self", "__conflicting_attrs", "jmespath_filters":
		return false
	}
	for _, pair := range policyRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range policyRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

func recordListControlError(key string) error {
	switch key {
	case "cls", "resource_type", "session", "self":
		return findInvalid("%q is a Resource/Proxy control; use the dedicated owned options for headers and microversion", key)
	}
	return nil
}

// WithRecordListQuery snapshots known kwargs in insertion order; unknowns are
// discarded before encoding their transport value, matching Source list.
// Declared query names and wire names go to the server; other declared Body
// attributes become local filters on each received row. Source runtime controls
// fail by presence before invoking an application JSON marshaler.
func WithRecordListQuery(key string, value any) RecordListOption {
	if recordListIgnores(key) {
		return func(*RecordListOpts) error { return nil }
	}
	if controlErr := recordListControlError(key); controlErr != nil {
		return func(*RecordListOpts) error { return controlErr }
	}
	var raw json.RawMessage
	var err error
	if key == "jmespath_filters" {
		switch v := value.(type) {
		case string:
			raw, err = json.Marshal(v)
		case json.RawMessage:
			raw = bytes.Clone(v)
		default:
			raw = json.RawMessage("null") // Source ignores every non-string, even truthy values.
		}
	} else {
		raw, err = json.Marshal(value)
	}
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		switch key {
		case "headers", "microversion", "base_path":
			formal, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{key: owned}})
			if err != nil {
				return err
			}
			switch key {
			case "headers":
				c.Headers = formal.Headers
			case "microversion":
				c.Microversion = formal.Microversion
			case "base_path":
				c.BasePath = formal.BasePath
			}
			delete(c.Query, key)
			c.order = slices.DeleteFunc(c.order, func(existing string) bool { return existing == key })
			return nil
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error {
		c.Headers = maps.Clone(owned)
		delete(c.Query, "headers")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "headers" })
		return nil
	}
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error {
		x := v
		c.Microversion = &x
		delete(c.Query, "microversion")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "microversion" })
		return nil
	}
}

// WithRecordListBasePath uses a service-relative collection route while
// retaining source, origin and continuation guards. Empty selects the default.
func WithRecordListBasePath(path string) RecordListOption {
	return func(c *RecordListOpts) error {
		value := path
		c.BasePath = &value
		delete(c.Query, "base_path")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "base_path" })
		return nil
	}
}

// WithRecordListConflictingAttrs restores Source kwargs once, preserving JSON
// object order. Duplicate bound paginated/base_path arguments remain errors.
func WithRecordListConflictingAttrs(value json.RawMessage) RecordListOption {
	return WithRecordListQuery("__conflicting_attrs", value)
}

func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMarker(v string) RecordListOption  { return WithRecordListQuery("marker", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }

// WithRecordListJMESPath evaluates the deprecated Proxy expression through
// ProxyListRecords. Empty expressions leave the lazy resource generator intact.
func WithRecordListJMESPath(expression string) RecordListOption {
	return WithRecordListQuery("jmespath_filters", expression)
}

func (a *API) policyRecordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("configured VPN policy API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, policyListProfile(), func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected VPN policy API client changed")
		}
		return nil
	})
}
func policyListRecord(value *neutronfind.Record) *IPsecPolicyRecord {
	if value == nil {
		return nil
	}
	return &IPsecPolicyRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
}
func applyRecordListOptions(ctx context.Context, engine *neutronfind.Engine, options []RecordListOption) (RecordListOpts, error) {
	policy := RecordListOpts{}
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, findInvalid("nil VPN policy list option")
		}
		next := copyRecordListOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = copyRecordListOpts(next)
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(recordListOptions(policy))
	if err != nil {
		return policy, err
	}
	policy.Query, policy.Headers, policy.Microversion, policy.BasePath, policy.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.BasePath, normalized.Order
	policy.expression = bytes.Clone(expression)
	return copyRecordListOpts(policy), nil
}

// recordListExpression removes the Proxy argument before Resource.list sees
// kwargs. Source evaluates only a truthy Python string; non-strings are ignored.
func recordListExpression(policy *RecordListOpts) (string, error) {
	raw := bytes.Clone(policy.expression)
	if raw == nil {
		return "", nil
	}
	policy.expression = nil
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return "", errors.Join(findInvalid("jmespath_filters must contain complete UTF-8 JSON"), err)
	}
	if !truthy {
		return "", nil
	}
	var expression string
	if err := json.Unmarshal(raw, &expression); err != nil {
		return "", nil
	}
	return expression, nil
}

// ListRecords is the lazy Source-shaped VPN policy list (Proxy.vpn_ipsec_policies). It has no
// implicit name hint or member fetch. Declared QueryParameters go to the
// server, other declared Body attributes filter received rows locally, and
// vpn_ipsec_policies_links, top-level next, HTTP Link and limit/marker continuation follow
// the pinned Resource.list paging.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*IPsecPolicyRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*IPsecPolicyRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListRecords", "vpn_ipsec_policies", cloudread.ContextError(ctx, err))
		}
		engine, err := a.policyRecordEngine(ctx)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		policy, err := applyRecordListOptions(ctx, engine, snapshot)
		if err == nil {
			var expression string
			expression, err = recordListExpression(&policy)
			if err == nil && expression != "" {
				err = findInvalid("JMESPath may return arbitrary values; use ProxyListRecords")
			}
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		for value, err := range engine.List(ctx, recordListOptions(policy)) {
			if !yield(policyListRecord(value), wrap(err)) {
				return
			}
		}
	}
}

// AllRecords discards partial collected values on error and returns a nonnil
// empty slice on success. Iterator consumers keep already yielded records.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*IPsecPolicyRecord, error) {
	values := make([]*IPsecPolicyRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
