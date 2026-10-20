package servers

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordFindOpts owns Source find_server defaults. Query, Headers, Microversion,
// Location and list caps are explicit Go extensions; the Python proxy exposes
// only identity, IgnoreMissing, Details and AllProjects.
type RecordFindOpts struct {
	IgnoreMissing, Details, AllProjects, Paginated *bool
	Query                                          map[string]json.RawMessage
	Headers                                        map[string]string
	Microversion                                   *string
	Location                                       *resource.CloudLocation
	Limit                                          *int
	Marker                                         string
	MaxItems                                       int
	order                                          []string
}
type RecordFindOption func(*RecordFindOpts) error

func serverFindCopy(v RecordFindOpts) RecordFindOpts {
	v.Query = serverFindClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	v.order = slices.Clone(v.order)
	for _, p := range []**bool{&v.IgnoreMissing, &v.Details, &v.AllProjects, &v.Paginated} {
		if *p != nil {
			x := **p
			*p = &x
		}
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := serverFindCopy(v)
	return func(c *RecordFindOpts) error { *c = serverFindCopy(owned); return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindDetails(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Details = &x; return nil }
}
func WithRecordFindAllProjects(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.AllProjects = &x; return nil }
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

// WithRecordFindMicroversion is an inherited Resource.find convenience: its
// explicit value applies to member GET; fallback uses the captured default.
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := maps.Clone(v)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordFindLocation(v resource.CloudLocation) RecordFindOption {
	owned := v.Clone()
	return func(c *RecordFindOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}
func WithRecordFindQuery(key string, v any) RecordFindOption {
	raw, err := json.Marshal(v)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if _, ok := c.Query[key]; !ok {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func serverFindKeys(v RecordFindOpts) []string {
	keys := make([]string, 0, len(v.Query))
	seen := make(map[string]bool)
	for _, key := range v.order {
		if _, ok := v.Query[key]; ok && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	var missing []string
	for key := range v.Query {
		if !seen[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return append(keys, missing...)
}

var serverFindQueryPairs = []struct{ name, wire string }{
	{"limit", "limit"},
	{"marker", "marker"},
	{"auto_disk_config", "auto_disk_config"},
	{"availability_zone", "availability_zone"},
	{"created_at", "created_at"},
	{"description", "description"},
	{"flavor", "flavor"},
	{"hostname", "hostname"},
	{"image", "image"},
	{"kernel_id", "kernel_id"},
	{"key_name", "key_name"},
	{"launch_index", "launch_index"},
	{"launched_at", "launched_at"},
	{"locked", "locked"},
	{"locked_by", "locked_by"},
	{"name", "name"},
	{"node", "node"},
	{"power_state", "power_state"},
	{"progress", "progress"},
	{"project_id", "project_id"},
	{"ramdisk_id", "ramdisk_id"},
	{"reservation_id", "reservation_id"},
	{"root_device_name", "root_device_name"},
	{"status", "status"},
	{"task_state", "task_state"},
	{"terminated_at", "terminated_at"},
	{"user_id", "user_id"},
	{"vm_state", "vm_state"},
	{"sort_key", "sort_key"},
	{"sort_dir", "sort_dir"},
	{"pinned_availability_zone", "pinned_availability_zone"},
	{"access_ipv4", "access_ip_v4"},
	{"access_ipv6", "access_ip_v6"},
	{"has_config_drive", "config_drive"},
	{"deleted_only", "deleted"},
	{"compute_host", "host"},
	{"is_soft_deleted", "soft_deleted"},
	{"ipv4_address", "ip"},
	{"ipv6_address", "ip6"},
	{"changes_since", "changes-since"},
	{"changes_before", "changes-before"},
	{"id", "uuid"},
	{"all_projects", "all_tenants"},
	{"tags", "tags"},
	{"any_tags", "tags-any"},
	{"not_tags", "not-tags"},
	{"not_any_tags", "not-tags-any"},
}

func serverFindQueryKey(key string) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return serverFindInvalid("query key must be nonblank UTF-8 without controls")
	}
	switch key {
	case "ignore_missing", "name_or_id", "session", "cls", "id", "self", "connection", "_synchronized", "details", "all_projects", "base_path", "list_base_path", "resource_type", "__conflicting_attrs", "requires_id", "resource_response_key", "skip_cache", "error_message", "headers", "microversion", "paginated", "max_items", "location", "interface_ip", "private_v4", "private_v6", "public_v4", "public_v6":
		return serverFindInvalid("query %q is a positional/constructor/control argument; use concrete options", key)
	}
	return nil
}
func serverFindPrepareQuery(v RecordFindOpts, identity string) (url.Values, url.Values, map[string]json.RawMessage, map[string]json.RawMessage, rest.ListControl, error) {
	get, list := make(url.Values), make(url.Values)
	seed, filters := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, nil, nil, control, serverFindInvalid("limit/cap must be nonnegative")
	}
	for _, key := range serverFindKeys(v) {
		raw := v.Query[key]
		if err := serverFindQueryKey(key); err != nil {
			return nil, nil, nil, nil, control, err
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, nil, nil, control, serverFindInvalid("query %q must contain UTF-8 JSON", key)
		}
		if v.AllProjects != nil && strings.EqualFold(key, "all_tenants") {
			return nil, nil, nil, nil, control, serverFindInvalid("typed all-projects conflicts with raw cross-project query")
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		if len(values) > 0 {
			get[key] = slices.Clone(values)
		}
		declared := false
		for _, p := range serverFindQueryPairs {
			if p.name == key {
				declared = true
				break
			}
		}
		for _, f := range serverFindFields {
			if key == f.name || key == f.wire {
				seed[f.wire] = bytes.Clone(raw)
			}
			if key == f.name && !declared {
				filters[key] = bytes.Clone(raw)
			}
		}
	}
	for _, pair := range serverFindQueryPairs {
		raw, ok := v.Query[pair.name]
		if !ok {
			raw, ok = v.Query[pair.wire]
		}
		if !ok {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		delete(list, pair.wire)
		if len(values) > 0 {
			list[pair.wire] = slices.Clone(values)
		}
	}
	if _, ok := v.Query["name"]; !ok {
		list.Set("name", identity)
	}
	if v.AllProjects != nil && *v.AllProjects {
		list.Set("all_tenants", "True")
	}
	if v.Limit != nil {
		list.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) || strings.IndexFunc(v.Marker, unicode.IsControl) >= 0 {
			return nil, nil, nil, nil, control, serverFindInvalid("marker must be valid UTF-8")
		}
		list.Set("marker", v.Marker)
	}
	seed["id"], _ = json.Marshal(identity)
	if _, err := serverFindBody(seed, serverFindFields); err != nil {
		return nil, nil, nil, nil, control, err
	}
	return get, list, filters, seed, control, nil
}
func serverFindLocalMatch(actual, filter json.RawMessage) (bool, error) {
	if actual == nil {
		actual = json.RawMessage("null")
	}
	if bytes.HasPrefix(bytes.TrimSpace(filter), []byte("{")) {
		truth, err := serverFindTruthy(actual)
		if err != nil || !truth {
			return false, err
		}
		wanted, err := cloudfilter.ObjectMembers(filter)
		if err != nil {
			return false, err
		}
		if len(wanted) == 0 {
			return true, nil
		}
		members, err := cloudfilter.ObjectMembers(actual)
		if err != nil {
			return false, serverFindInvalid("truthy local value does not provide mapping.get")
		}
		fields := make(map[string]json.RawMessage)
		for _, m := range members {
			fields[m.Key] = m.Value
		}
		for _, m := range wanted {
			same, err := serverFindLocalMatch(fields[m.Key], m.Value)
			if err != nil || !same {
				return same, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}
