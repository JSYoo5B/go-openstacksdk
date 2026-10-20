package snapshots

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// RecordFindOpts defaults to details and ignore-missing enabled. Query and
// pagination controls are explicit Go extensions for the fallback list only.
type RecordFindOpts struct {
	IgnoreMissing, Details, AllProjects, Paginated *bool
	Query                                          map[string]json.RawMessage
	Headers                                        map[string]string
	Limit                                          *int
	Marker                                         string
	MaxItems                                       int
}
type RecordFindOption func(*RecordFindOpts) error

func copyFindOpts(v RecordFindOpts) RecordFindOpts {
	v.Query = findClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	for _, field := range []**bool{&v.IgnoreMissing, &v.Details, &v.AllProjects, &v.Paginated} {
		if *field != nil {
			x := **field
			*field = &x
		}
	}
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
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
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := maps.Clone(v)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func findQueryWire(key string) (string, bool) {
	switch key {
	case "name", "status", "volume_id", "project_id", "limit", "marker", "offset", "sort_dir", "sort_key", "sort", "all_tenants":
		return key, true
	case "all_projects":
		return "all_tenants", true
	default:
		return "", false
	}
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	_, server := findQueryWire(key)
	if !server && !findBodyKnown(key) {
		return func(*RecordFindOpts) error { return nil }
	}
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
func prepareFindQuery(v RecordFindOpts, identity string) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query, filters := make(url.Values), make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, control, findInvalid("negative fallback limit or cap")
	}
	for key, raw := range v.Query {
		if key == "id" {
			return nil, nil, control, findInvalid("id query conflicts with the find identity")
		}
		if key == "all_projects" || key == "all_tenants" {
			if v.AllProjects != nil {
				continue
			}
			if key == "all_tenants" {
				if _, client := v.Query["all_projects"]; client {
					continue
				}
			}
		}
		wire, server := findQueryWire(key)
		if !server && !findBodyKnown(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, control, findInvalid("query %q must be UTF-8 JSON", key)
		}
		if server {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, control, err
			}
			if len(values) != 0 {
				query[wire] = slices.Clone(values)
			}
		} else {
			filters[key] = bytes.Clone(raw)
		}
	}
	if _, exists := v.Query["name"]; !exists {
		query.Set("name", identity)
	}
	if v.AllProjects != nil && *v.AllProjects {
		query.Set("all_tenants", "True")
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, control, findInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	return query, filters, control, nil
}

// The dictionary predicate follows Source _dict_filter, including truthy
// scalar+empty-object matches; nonempty object predicates require mapping.get.
func findLocalValue(actual, filter json.RawMessage) (bool, error) {
	if actual == nil {
		actual = json.RawMessage("null")
	}
	if bytes.HasPrefix(bytes.TrimSpace(filter), []byte("{")) {
		truthy, err := cloudfilter.PythonTruthy(actual)
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
			return false, findInvalid("truthy filter value has no mapping get: %v", err)
		}
		fields := make(map[string]json.RawMessage, len(observed))
		for _, member := range observed {
			fields[member.Key] = member.Value
		}
		for _, member := range wanted {
			matched, err := findLocalValue(fields[member.Key], member.Value)
			if err != nil || !matched {
				return matched, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}
