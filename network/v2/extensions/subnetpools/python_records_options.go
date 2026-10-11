package subnetpools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// RecordMutationOpts owns only supplied Body attrs plus explicit Go request
// conveniences. No attrs is a no-HTTP commit. is_default is not synthesized;
// fixed stock Neutron policy requires admin to mutate that attribute.
type RecordMutationOpts struct {
	Fields       map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type RecordMutationOption func(*RecordMutationOpts) error

func copyPoolMutation(v RecordMutationOpts) RecordMutationOpts {
	v.Fields = poolRecordClone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	v.order = slices.Clone(v.order)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordMutationOpts(v RecordMutationOpts) RecordMutationOption {
	owned := copyPoolMutation(v)
	return func(c *RecordMutationOpts) error { *c = copyPoolMutation(owned); return nil }
}

// WithRecordAttribute snapshots raw canonical/wire Body attrs; unknown fields,
// if_revision/if_match and explicit null remain ordinary Body data here.
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
		if _, present := c.Fields[key]; !present {
			c.order = append(c.order, key)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordMutationHeaders(v map[string]string) RecordMutationOption {
	owned := maps.Clone(v)
	return func(c *RecordMutationOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordMutationMicroversion(v string) RecordMutationOption {
	return func(c *RecordMutationOpts) error { x := v; c.Microversion = &x; return nil }
}
func poolRecordOrderedKeys(fields map[string]json.RawMessage, order []string) []string {
	keys := make([]string, 0, len(fields))
	seen := make(map[string]bool)
	for _, key := range order {
		if _, present := fields[key]; present && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	var other []string
	for key := range fields {
		if !seen[key] {
			other = append(other, key)
		}
	}
	sort.Strings(other)
	return append(keys, other...)
}
func poolRecordName(key string) error {
	if key == "" || !utf8.ValidString(key) {
		return poolRecordInvalid("nonempty UTF-8 attribute/query name required")
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return poolRecordInvalid("attribute/query name has controls")
		}
	}
	return nil
}
func poolRecordMutation(v RecordMutationOpts) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	wire, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	for _, key := range poolRecordOrderedKeys(v.Fields, v.order) {
		if err := poolRecordName(key); err != nil {
			return nil, nil, err
		}
		switch key {
		case "id", "self", "cls", "subnet_pool", "value", "resource_type", "base_path", "microversion", "connection", "_synchronized", "__conflicting_attrs", "location":
			return nil, nil, poolRecordInvalid("%q is an SDK-owned update identity or Resource control", key)
		}
		raw := v.Fields[key]
		if err := poolRecordJSON(raw); err != nil {
			return nil, nil, poolRecordInvalid("Body %q: %v", key, err)
		}
		canonical := poolRecordCanonical(key)
		destination := canonical
		if name, known := neutronBodyFieldNames[canonical]; known {
			destination = name
		}
		wire[destination] = bytes.Clone(raw)
		seed[canonical] = bytes.Clone(raw)
	}
	return wire, seed, nil
}

// RecordListOpts preserves Source kwargs values. Typed pagination controls
// apply to the list only; unknown list kwargs are discarded by Source mapping.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Limit        *int
	Marker       string
	MaxItems     int
	Paginated    *bool
	order        []string
}
type RecordListOption func(*RecordListOpts) error

func copyPoolList(v RecordListOpts) RecordListOpts {
	v.Query = poolRecordClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	v.order = slices.Clone(v.order)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
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
	owned := copyPoolList(v)
	return func(c *RecordListOpts) error { *c = copyPoolList(owned); return nil }
}
func WithRecordListQuery(key string, value any) RecordListOption {
	keyErr := poolRecordQueryKey(key, false)
	_, server := poolRecordQueryWire(key)
	if keyErr != nil || !server && !poolRecordKnown(key) {
		return func(*RecordListOpts) error { return keyErr }
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
		if _, present := c.Query[key]; !present {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
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

// RecordFindOpts defaults to ignore-missing/full pagination. Original Query
// kwargs are member query/constructor seed; fallback alone transposes aliases.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	Limit         *int
	Marker        string
	MaxItems      int
	Paginated     *bool
	order         []string
}
type RecordFindOption func(*RecordFindOpts) error

func poolFindList(v RecordFindOpts) RecordListOpts {
	return RecordListOpts{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Limit: v.Limit, Marker: v.Marker, MaxItems: v.MaxItems, Paginated: v.Paginated, order: v.order}
}
func copyPoolFind(v RecordFindOpts) RecordFindOpts {
	list := copyPoolList(poolFindList(v))
	v.Query, v.Headers, v.Microversion, v.Limit, v.Marker, v.MaxItems, v.Paginated, v.order = list.Query, list.Headers, list.Microversion, list.Limit, list.Marker, list.MaxItems, list.Paginated, list.order
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyPoolFind(v)
	return func(c *RecordFindOpts) error { *c = copyPoolFind(owned); return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
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
		if _, present := c.Query[key]; !present {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := maps.Clone(v)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFindLimit(v int) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Limit = &x; return nil }
}
func WithRecordFindMarker(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { c.Marker = v; return nil }
}
func WithRecordFindMaxItems(v int) RecordFindOption {
	return func(c *RecordFindOpts) error { c.MaxItems = v; return nil }
}
func WithRecordFindPaginated(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Paginated = &x; return nil }
}

var poolRecordQueryAliases = map[string]string{"is_shared": "shared", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any"}

func poolRecordQueryWire(key string) (string, bool) {
	if wire, known := poolRecordQueryAliases[key]; known {
		return wire, true
	}
	for _, wire := range poolRecordQueryAliases {
		if key == wire {
			return wire, true
		}
	}
	switch key {
	case "limit", "marker", "address_scope_id", "description", "fields", "ip_version", "is_default", "name", "project_id", "sort_key", "sort_dir", "tags":
		return key, true
	}
	return "", false
}
func poolRecordQueryKey(key string, find bool) error {
	if err := poolRecordName(key); err != nil {
		return err
	}
	switch key {
	case "self", "cls", "session", "name_or_id", "ignore_missing", "connection", "_synchronized", "base_path", "list_base_path", "resource_type", "__conflicting_attrs", "requires_id", "resource_response_key", "skip_cache", "error_message", "headers", "microversion", "paginated", "max_items", "allow_unknown_params", "jmespath_filters":
		return poolRecordInvalid("%q is a Resource/control argument; use concrete options", key)
	case "id":
		if find {
			return poolRecordInvalid("id kwargs collide with the find identity")
		}
	}
	return nil
}
func poolRecordQuery(v RecordListOpts, find bool, identity string) (url.Values, url.Values, map[string]json.RawMessage, map[string]json.RawMessage, rest.ListControl, error) {
	member, list := make(url.Values), make(url.Values)
	filters, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, nil, nil, control, poolRecordInvalid("negative list limit/cap")
	}
	for _, key := range poolRecordOrderedKeys(v.Query, v.order) {
		if err := poolRecordQueryKey(key, find); err != nil {
			return nil, nil, nil, nil, control, err
		}
		if key == "all_projects" {
			continue
		}
		wire, server := poolRecordQueryWire(key)
		if !find && !server && !poolRecordKnown(key) {
			continue
		}
		raw := v.Query[key]
		if err := poolRecordJSON(raw); err != nil {
			return nil, nil, nil, nil, control, poolRecordInvalid("query %q: %v", key, err)
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		if find {
			if len(values) > 0 {
				member[key] = slices.Clone(values)
			}
			if key != "location" {
				seed[poolRecordCanonical(key)] = bytes.Clone(raw)
			}
		}
		if server {
			overridden := false
			for canonical, alias := range poolRecordQueryAliases {
				if key == alias {
					_, overridden = v.Query[canonical]
					if overridden {
						break
					}
				}
			}
			if overridden {
				continue
			}
			if len(values) > 0 {
				list[wire] = slices.Clone(values)
			}
		} else if poolRecordKnown(key) {
			filters[key] = bytes.Clone(raw)
		}
	}
	if find {
		if _, present := v.Query["name"]; !present {
			list.Set("name", identity)
		}
		seed["id"], _ = json.Marshal(identity)
		if err := poolRecordProject(&SubnetPoolRecord{}, seed); err != nil {
			return nil, nil, nil, nil, control, err
		}
	}
	if v.Limit != nil {
		list.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, nil, nil, control, poolRecordInvalid("marker must be UTF-8")
		}
		list.Set("marker", v.Marker)
	}
	return member, list, filters, seed, control, nil
}
func poolRecordLocalMatch(actual, wanted json.RawMessage) (bool, error) {
	if actual == nil {
		actual = json.RawMessage("null")
	}
	if bytes.HasPrefix(bytes.TrimSpace(wanted), []byte("{")) {
		truth, err := poolRecordBoolean(actual)
		if err != nil || !bytes.Equal(bytes.TrimSpace(truth), []byte("true")) {
			return false, err
		}
		keys, err := cloudfilter.ObjectMembers(wanted)
		if err != nil {
			return false, err
		}
		if len(keys) == 0 {
			return true, nil
		}
		observed, err := cloudfilter.ObjectMembers(actual)
		if err != nil {
			return false, poolRecordInvalid("truthy local filter has no mapping.get: %v", err)
		}
		values := make(map[string]json.RawMessage, len(observed))
		for _, member := range observed {
			values[member.Key] = member.Value
		}
		for _, member := range keys {
			equal, err := poolRecordLocalMatch(values[member.Key], member.Value)
			if err != nil || !equal {
				return equal, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, wanted)
}
func poolRecordOptions[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, base T, copy func(T) T, options []O) (T, error) {
	config := copy(base)
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, poolRecordInvalid("nil record option")
		}
		next := copy(config)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return config, err
		}
		config = copy(next)
	}
	return config, guard(ctx)
}
