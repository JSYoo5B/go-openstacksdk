package rules

import (
	"bytes"
	"encoding/json"
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

// RecordFindOpts defaults to ignore-missing and full pagination. Query snapshots
// the Source kwargs used by the member fetch, constructor and fallback list.
// The typed list controls affect fallback only. Headers apply to both phases;
// Microversion overrides member fetch only, as in Source Resource.find.
type RecordFindOpts struct {
	IgnoreMissing, Paginated *bool
	Query                    map[string]json.RawMessage
	Headers                  map[string]string
	Microversion             *string
	Limit                    *int
	Marker                   string
	MaxItems                 int
	order                    []string
}
type RecordFindOption func(*RecordFindOpts) error

func copyFindOpts(v RecordFindOpts) RecordFindOpts {
	v.Query = findClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	v.order = slices.Clone(v.order)
	for _, p := range []**bool{&v.IgnoreMissing, &v.Paginated} {
		if *p != nil {
			x := **p
			*p = &x
		}
	}
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
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
func WithRecordFindPaginated(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Paginated = &x; return nil }
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
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := maps.Clone(v)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// WithRecordFindQuery records insertion order for Source constructor aliases.
// Unknown kwargs remain member query/seed values and are dropped from LIST.
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
		if _, exists := c.Query[key]; !exists {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func findKeys(v RecordFindOpts) []string {
	keys := make([]string, 0, len(v.Query))
	seen := make(map[string]bool)
	for _, key := range v.order {
		if _, exists := v.Query[key]; exists && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	var remaining []string
	for key := range v.Query {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	return append(keys, remaining...)
}

var findQueryAliases = map[string]string{"ether_type": "ethertype", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any"}

func findQueryWire(key string) (string, bool) {
	if wire, ok := findQueryAliases[key]; ok {
		return wire, true
	}
	for _, wire := range findQueryAliases {
		if wire == key {
			return wire, true
		}
	}
	switch key {
	case "limit", "marker", "description", "direction", "fields", "id", "protocol", "remote_group_id", "security_group_id", "remote_address_group_id", "port_range_max", "port_range_min", "remote_ip_prefix", "revision_number", "project_id", "tenant_id", "sort_dir", "sort_key", "tags":
		return key, true
	}
	return "", false
}
func findQueryKey(key string) error {
	if key == "" || !utf8.ValidString(key) {
		return findInvalid("query name must be nonempty UTF-8")
	}
	for _, r := range key {
		if unicode.IsControl(r) {
			return findInvalid("query name has controls")
		}
	}
	switch key {
	case "ignore_missing", "name_or_id", "session", "cls", "id", "self", "connection", "_synchronized", "base_path", "list_base_path", "resource_type", "__conflicting_attrs", "requires_id", "resource_response_key", "skip_cache", "error_message", "headers", "microversion", "paginated", "max_items":
		return findInvalid("query %q is a Resource/control argument; use the supported concrete options", key)
	}
	return nil
}
func prepareFindQuery(v RecordFindOpts, identity string) (url.Values, url.Values, map[string]json.RawMessage, map[string]json.RawMessage, json.RawMessage, rest.ListControl, error) {
	member, list := make(url.Values), make(url.Values)
	filters, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	var match json.RawMessage
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, nil, nil, nil, control, findInvalid("negative list limit or cap")
	}
	for _, key := range findKeys(v) {
		raw := v.Query[key]
		if err := findQueryKey(key); err != nil {
			return nil, nil, nil, nil, nil, control, err
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, nil, nil, nil, control, findInvalid("query %q must contain UTF-8 JSON", key)
		}
		// Resource.find consumes all_projects separately; neither Neutron class has
		// a corresponding LIST mapping. It does not broaden tenant authorization.
		if key == "all_projects" {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, nil, control, err
		}
		if len(values) > 0 {
			member[key] = slices.Clone(values)
		}
		if findHeaderKey(key) {
			match, err = findList(raw)
			if err != nil {
				return nil, nil, nil, nil, nil, control, err
			}
		} else if key != "location" {
			canonical, _ := findCanonical(key)
			seed[canonical] = bytes.Clone(raw)
		}
		wire, server := findQueryWire(key)
		if server {
			overridden := false
			for client, alias := range findQueryAliases {
				if key == alias {
					if _, exists := v.Query[client]; exists {
						overridden = true
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
		} else if _, declared := neutronBodyFieldNames[key]; declared {
			filters[key] = bytes.Clone(raw)
		}
	}
	// Rule name is an inherited Body field, not a server query or automatic hint.
	if v.Limit != nil {
		list.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, nil, nil, nil, control, findInvalid("marker must be UTF-8")
		}
		list.Set("marker", v.Marker)
	}
	seed["id"], _ = json.Marshal(identity)
	// cls.existing builds its descriptor dictionary before issuing member HTTP.
	if err := findProject(&SecurityGroupRuleRecord{}, seed, match); err != nil {
		return nil, nil, nil, nil, nil, control, err
	}
	return member, list, filters, seed, match, control, nil
}
func findLocalValue(actual, filter json.RawMessage) (bool, error) {
	if actual == nil {
		actual = json.RawMessage("null")
	}
	if bytes.HasPrefix(bytes.TrimSpace(filter), []byte("{")) {
		truthy, err := findTruthy(actual)
		if err != nil || !truthy {
			return false, err
		}
		wanted, err := cloudfilter.ObjectMembers(filter)
		if err != nil {
			return false, err
		}
		if len(wanted) == 0 {
			return true, nil
		}
		observed, err := cloudfilter.ObjectMembers(actual)
		if err != nil {
			return false, findInvalid("truthy local filter value has no mapping.get: %v", err)
		}
		fields := make(map[string]json.RawMessage, len(observed))
		for _, member := range observed {
			fields[member.Key] = member.Value
		}
		for _, member := range wanted {
			equal, err := findLocalValue(fields[member.Key], member.Value)
			if err != nil || !equal {
				return equal, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}
