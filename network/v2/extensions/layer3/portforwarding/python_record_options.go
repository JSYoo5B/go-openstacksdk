package portforwarding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// RecordMutationOpts owns only supplied Body changes. PortForwarding inherits
// Resource, not NetworkResource: if_match/if_revision are ordinary unknown Body
// extensions here. Neither option silently introduces conditional headers.
type RecordMutationOpts struct{ Fields map[string]json.RawMessage }
type RecordMutationOption func(*RecordMutationOpts) error

func copyForwardingMutation(v RecordMutationOpts) RecordMutationOpts {
	v.Fields = forwardingClone(v.Fields)
	return v
}
func WithRecordMutationOpts(v RecordMutationOpts) RecordMutationOption {
	owned := copyForwardingMutation(v)
	return func(c *RecordMutationOpts) error { *c = copyForwardingMutation(owned); return nil }
}
func WithRecordAttribute(key string, value any) RecordMutationOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordMutationOpts) error {
		if err != nil {
			return err
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}
func forwardingBody(v RecordMutationOpts, update bool) (map[string]json.RawMessage, error) {
	body := forwardingClone(v.Fields)
	for key, raw := range body {
		switch key {
		case "floatingip_id", "base_path", "microversion", "connection", "_synchronized", "resource_type", "__conflicting_attrs", "location", "self":
			return nil, forwardingInvalid("%q is an SDK-owned scope or Resource control", key)
		}
		if update && key == "id" {
			return nil, forwardingInvalid("update ID is fixed by the method argument")
		}
		if !utf8.ValidString(key) || strings.ContainsAny(key, "\r\n\x00") || !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, forwardingInvalid("Body %q must contain UTF-8 JSON", key)
		}
	}
	return body, nil
}

type RecordReadOpts struct{ Fields []string }
type RecordReadOption func(*RecordReadOpts) error

func copyForwardingRead(v RecordReadOpts) RecordReadOpts {
	v.Fields = append([]string(nil), v.Fields...)
	return v
}

// WithRecordFields preserves unknown and repeated field names and their order.
// This Resource's canonical Body names already equal their wire names.
func WithRecordFields(fields ...string) RecordReadOption {
	owned := append([]string(nil), fields...)
	return func(c *RecordReadOpts) error { c.Fields = append([]string(nil), owned...); return nil }
}

type RecordDeleteOpts struct{ IgnoreMissing *bool }
type RecordDeleteOption func(*RecordDeleteOpts) error

func copyForwardingDelete(v RecordDeleteOpts) RecordDeleteOpts {
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithRecordDeleteIgnoreMissing(v bool) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

type RecordListOpts struct {
	Query     map[string]json.RawMessage
	Headers   map[string]string
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
}
type RecordListOption func(*RecordListOpts) error

func copyForwardingList(v RecordListOpts) RecordListOpts {
	v.Query = forwardingClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyForwardingList(v)
	return func(c *RecordListOpts) error { *c = copyForwardingList(owned); return nil }
}

// WithRecordListQuery sends declared query parameters, filters other declared
// Body fields locally, and ignores unknown kwargs exactly as Resource.list.
func WithRecordListQuery(key string, value any) RecordListOption {
	if !forwardingKnown(key) && !forwardingServerQuery(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordListLimit(v int) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Limit = &x; return nil }
}
func WithRecordListMarker(v string) RecordListOption {
	return func(c *RecordListOpts) error { c.Marker = v; return nil }
}
func WithRecordListMaxItems(v int) RecordListOption {
	return func(c *RecordListOpts) error { c.MaxItems = v; return nil }
}
func WithRecordListPaginated(v bool) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Paginated = &x; return nil }
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func forwardingServerQuery(key string) bool {
	switch key {
	case "fields", "internal_port_id", "external_port", "protocol", "sort_key", "sort_dir", "limit", "marker":
		return true
	}
	return false
}
func forwardingList(v RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query := make(url.Values)
	filters := make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, forwardingInvalid("negative limit or max items")
	}
	for key, raw := range v.Query {
		if !forwardingKnown(key) && !forwardingServerQuery(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, forwardingInvalid("query %q must be UTF-8 JSON", key)
		}
		if forwardingServerQuery(key) {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, rest.ListControl{}, err
			}
			if len(values) != 0 {
				query[key] = values
			}
		} else {
			filters[key] = bytes.Clone(raw)
		}
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, rest.ListControl{}, forwardingInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}

type RecordFindOpts struct {
	IgnoreMissing *bool
	Query         map[string]json.RawMessage
}
type RecordFindOption func(*RecordFindOpts) error

func copyForwardingFind(v RecordFindOpts) RecordFindOpts {
	v.Query = forwardingClone(v.Query)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyForwardingFind(v)
	return func(c *RecordFindOpts) error { *c = copyForwardingFind(owned); return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

// WithRecordFindQuery preserves the Source difference between its unfiltered
// fetch kwargs and the fallback's declared server/local query projections.
func WithRecordFindQuery(key string, value any) RecordFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func forwardingFind(v RecordFindOpts, parent string) (url.Values, RecordListOpts, error) {
	direct := make(url.Values)
	direct.Set("floatingip_id", parent)
	list := RecordListOpts{Query: make(map[string]json.RawMessage)}
	for key, raw := range v.Query {
		switch key {
		case "floatingip_id", "base_path", "microversion", "list_base_path":
			return nil, list, forwardingInvalid("find scope and service route are fixed: %q", key)
		}
		if !utf8.ValidString(key) || !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, list, forwardingInvalid("find query %q must be UTF-8 JSON", key)
		}
		// Resource.find consumes all_projects before its direct fetch and passes it
		// only to list. This Resource has no such server or Body filter, so ignored.
		if key == "all_projects" {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, list, err
		}
		if len(values) != 0 {
			direct[key] = values
		}
		switch key {
		case "paginated":
			var value bool
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, list, forwardingInvalid("paginated requires bool")
			}
			list.Paginated = &value
		case "max_items":
			if err := json.Unmarshal(raw, &list.MaxItems); err != nil {
				return nil, list, forwardingInvalid("max_items requires integer")
			}
		case "headers":
			if err := json.Unmarshal(raw, &list.Headers); err != nil {
				return nil, list, forwardingInvalid("headers require a string map")
			}
		default:
			list.Query[key] = bytes.Clone(raw)
		}
	}
	return direct, list, nil
}
func applyForwardingOptions[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, forwardingInvalid("nil record option")
		}
		next := copyValue(value)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}
