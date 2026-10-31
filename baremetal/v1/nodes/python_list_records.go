package nodes

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

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

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

// RecordListOption is one option of ListRecords, AllRecords and
// ProxyListRecords: WithRecordListOpts, WithRecordListDetails,
// WithRecordListJMESPath, WithRecordListConflictingAttrs, request.Custom over
// *RecordListOpts or a shared call. request.Query snapshots one Source kwarg
// (see recordListKeyword); request.Limit, request.Marker, request.MaxItems and
// request.Paginated are the limit, marker, max_items and paginated kwargs;
// request.Header adds and request.Headers replaces the additional headers;
// request.Microversion and request.BasePath select the per-page microversion
// and a service-relative collection route while retaining source, origin and
// continuation guards (an empty base path selects the default). A nil option
// fails with resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListBasePathCall

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
	return request.Custom(func(c *RecordListOpts) error { *c = copyRecordListOpts(owned); return nil })
}

// recordListIgnores reports kwargs that Source Resource.list discards before
// value evaluation: neither a declared query name/wire nor a Body attribute.
func recordListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return false
	}
	switch key {
	case "details", "cls", "paginated", "max_items", "base_path", "resource_type", "session", "microversion", "headers", "self", "__conflicting_attrs", "jmespath_filters":
		return false
	}
	for _, pair := range listRecordProfile().Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range listRecordProfile().Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

func recordListControlError(key string) error {
	switch key {
	case "cls", "resource_type", "session", "self":
		return listRecordInvalid("%q is a Resource/Proxy control; use the dedicated owned options for headers and microversion", key)
	}
	return nil
}

// recordListKeyword snapshots one known kwarg (request.Query and the paging
// calls) in insertion order; unknowns are discarded before their marshal
// error is reported, matching Source list. Declared query names and wire
// names go to the server; other declared Body attributes become local filters
// on each received row. Source runtime controls fail by presence.
// jmespath_filters keeps a JSON string expression; Source ignores every
// non-string value.
func recordListKeyword(c *RecordListOpts, key string, raw json.RawMessage, err error) error {
	if recordListIgnores(key) {
		return nil
	}
	if controlErr := recordListControlError(key); controlErr != nil {
		return controlErr
	}
	if err != nil {
		return err
	}
	owned := bytes.Clone(raw)
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
		dropRecordListKeyword(c, key)
		return nil
	}
	if c.Query == nil {
		c.Query = make(map[string]json.RawMessage)
	}
	if !slices.Contains(c.order, key) {
		c.order = append(c.order, key)
	}
	c.Query[key] = owned
	return nil
}

func dropRecordListKeyword(c *RecordListOpts, key string) {
	delete(c.Query, key)
	c.order = slices.DeleteFunc(c.order, func(existing string) bool { return existing == key })
}

// recordListCalls interprets shared calls over RecordListOpts.
var recordListCalls = calls.Spec[RecordListOpts]{
	Header: func(c *RecordListOpts, key, value string) error {
		headers := maps.Clone(c.Headers)
		if headers == nil {
			headers = make(map[string]string)
		}
		headers[key] = value
		c.Headers = headers
		dropRecordListKeyword(c, "headers")
		return nil
	},
	Headers: func(c *RecordListOpts, values map[string]string) error {
		c.Headers = values
		dropRecordListKeyword(c, "headers")
		return nil
	},
	Microversion: func(c *RecordListOpts, version string) error {
		c.Microversion = &version
		dropRecordListKeyword(c, "microversion")
		return nil
	},
	BasePath: func(c *RecordListOpts, path string) error {
		c.BasePath = &path
		dropRecordListKeyword(c, "base_path")
		return nil
	},
	Query:  calls.JSONQuery(recordListKeyword),
	Paging: calls.JSONPaging(recordListKeyword),
}

// WithRecordListConflictingAttrs restores Source kwargs once, preserving JSON
// object order. Duplicate bound paginated/base_path arguments remain errors.
func WithRecordListConflictingAttrs(value json.RawMessage) RecordListOption {
	return request.Query("__conflicting_attrs", value)
}

// WithRecordListJMESPath evaluates the deprecated Proxy expression through
// ProxyListRecords. Empty expressions leave the lazy resource generator intact.
func WithRecordListJMESPath(expression string) RecordListOption {
	return request.Query("jmespath_filters", expression)
}

func (a *API) listRecordRecordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, listRecordInvalid("configured Node API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, listRecordProfile(), func(context.Context) error {
		if a.client != selected {
			return listRecordInvalid("selected Node API client changed")
		}
		return nil
	})
}

func listRecordListRecord(value *neutronfind.Record) *NodeRecord {
	if value == nil {
		return nil
	}
	return &NodeRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
}
func applyRecordListOptions(ctx context.Context, engine *neutronfind.Engine, options []RecordListOption) (RecordListOpts, error) {
	policy := RecordListOpts{}
	for _, apply := range calls.Translate(options, recordListCalls) {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, listRecordInvalid("nil Node list option")
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
	if err := normalizeNodeListPolicy(&policy); err != nil {
		return policy, err
	}
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
		return "", errors.Join(listRecordInvalid("jmespath_filters must contain complete UTF-8 JSON"), err)
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

// ListRecords is the lazy Source-shaped Node list (Proxy.nodes). It has no
// implicit name hint or member fetch. Declared QueryParameters go to the
// server, other declared Body attributes filter received rows locally, and
// nodes_links, top-level next, HTTP Link and limit/marker continuation follow
// the pinned Resource.list paging.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*NodeRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*NodeRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListRecords", "nodes", cloudread.ContextError(ctx, err))
		}
		engine, err := a.listRecordRecordEngine(ctx)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		client := nodeReadClientSnapshot(a)
		policy, err := applyRecordListOptions(ctx, engine, snapshot)
		if err == nil {
			var expression string
			expression, err = recordListExpression(&policy)
			if err == nil && expression != "" {
				err = listRecordInvalid("JMESPath may return arbitrary values; use ProxyListRecords")
			}
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		engine, discovery, err := a.versionedReadEngine(ctx, engine, policy.Microversion, client, policy.Headers)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		policy.Microversion = nil
		for value, err := range engine.List(ctx, recordListOptions(policy)) {
			row := listRecordListRecord(value)
			if row != nil {
				row.Discovery = cloneNodeDiscovery(discovery)
			}
			if !yield(row, wrap(err)) {
				return
			}
		}
	}
}

// AllRecords discards partial collected values on error and returns a nonnil
// empty slice on success. Iterator consumers keep already yielded records.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*NodeRecord, error) {
	values := make([]*NodeRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
