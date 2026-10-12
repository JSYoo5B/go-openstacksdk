package sfcrecord

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
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ReadOpts carries per-request headers, a Go extension to the Python call.
type ReadOpts struct{ Headers map[string]string }

// WriteOpts carries Python **attrs as raw JSON keyed by attribute name or
// Body key, plus per-request headers.
type WriteOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}

// DeleteOpts selects ignore_missing (nil means the Python default true).
type DeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// FindOpts selects ignore_missing (nil means the Python default true) and
// carries the Python find **query names as raw JSON.
type FindOpts struct {
	IgnoreMissing *bool
	Query         map[string]json.RawMessage
}

// ListOpts carries Python **query names as raw JSON, the limit and marker
// pagination parameters and the Resource.list max_items and paginated knobs.
type ListOpts struct {
	Headers   map[string]string
	Query     map[string]json.RawMessage
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
}

func copyBool(v *bool) *bool {
	if v == nil {
		return nil
	}
	x := *v
	return &x
}

func CopyRead(v ReadOpts) ReadOpts { v.Headers = maps.Clone(v.Headers); return v }
func CopyWrite(v WriteOpts) WriteOpts {
	v.Headers, v.Fields = maps.Clone(v.Headers), cloneFields(v.Fields)
	return v
}
func CopyDelete(v DeleteOpts) DeleteOpts {
	v.Headers, v.IgnoreMissing = maps.Clone(v.Headers), copyBool(v.IgnoreMissing)
	return v
}
func CopyFind(v FindOpts) FindOpts {
	v.IgnoreMissing, v.Query = copyBool(v.IgnoreMissing), cloneFields(v.Query)
	return v
}
func CopyList(v ListOpts) ListOpts {
	v.Headers, v.Query, v.Paginated = maps.Clone(v.Headers), cloneFields(v.Query), copyBool(v.Paginated)
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	return v
}

// Marshal snapshots one option value; marshal errors surface when applied.
func Marshal(value any) func() (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func() (json.RawMessage, error) { return bytes.Clone(owned), err }
}

// SetField stores a snapshotted value under key.
func SetField(target *map[string]json.RawMessage, key string, value func() (json.RawMessage, error)) error {
	raw, err := value()
	if err != nil {
		return err
	}
	if *target == nil {
		*target = make(map[string]json.RawMessage)
	}
	(*target)[key] = raw
	return nil
}

// Apply runs options in order on owned copies; a nil option fails before HTTP.
func Apply[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, Invalid("nil option")
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

// reserved names are Python Resource/Proxy keyword controls. Go exposes the
// supported ones as typed options, so they never travel as query names.
func reserved(key string) bool {
	switch key {
	case "self", "cls", "session", "name_or_id", "ignore_missing", "connection", "_synchronized",
		"base_path", "list_base_path", "resource_type", "requires_id", "resource_response_key",
		"skip_cache", "error_message", "headers", "microversion", "paginated", "max_items",
		"allow_unknown_params", "jmespath_filters", "location":
		return true
	}
	return false
}

func sortedKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// prepareList splits Python **query into server query values (Requests
// encoding) and local Body filters, and ignores unknown names.
func (s *Spec) prepareList(v ListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query := make(url.Values)
	filters := make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, Invalid("negative limit or maximum items")
	}
	for _, key := range sortedKeys(v.Query) {
		raw := v.Query[key]
		wire, isQuery := s.Query[key]
		field, isFilter := s.Filters[key]
		if !isQuery && !isFilter {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, Invalid("query %q must be UTF-8 JSON", key)
		}
		if isQuery {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, rest.ListControl{}, err
			}
			if len(values) != 0 {
				query[wire] = values
			}
			continue
		}
		filters[field] = bytes.Clone(raw)
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, rest.ListControl{}, Invalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	if query.Has("limit") {
		n, err := strconv.Atoi(query.Get("limit"))
		if len(query["limit"]) != 1 || err != nil || n < 0 {
			return nil, nil, rest.ListControl{}, Invalid("limit must be one nonnegative integer")
		}
	}
	if query.Has("marker") && (len(query["marker"]) != 1 || query.Get("marker") == "") {
		return nil, nil, rest.ListControl{}, Invalid("marker must be one nonempty value")
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}

// prepareFind splits Python find **query: every name goes to the member GET
// query (Resource.fetch **params), declared attributes seed the found
// resource (Resource.existing), and the list fallback receives the same
// query plus name=name_or_id unless the query names one.
func (s *Spec) prepareFind(values map[string]json.RawMessage, nameOrID string) (url.Values, map[string]json.RawMessage, ListOpts, error) {
	member := make(url.Values)
	seed := make(map[string]json.RawMessage)
	list := ListOpts{Query: make(map[string]json.RawMessage, len(values)+1)}
	if err := ValidateID("name or ID", nameOrID); err != nil {
		return nil, nil, list, err
	}
	for _, key := range sortedKeys(values) {
		raw := values[key]
		if key == "" || !utf8.ValidString(key) || reserved(key) {
			return nil, nil, list, Invalid("find query name %q is not a query attribute", key)
		}
		if key == "id" {
			return nil, nil, list, Invalid("find query id collides with the find identity")
		}
		if key == "all_projects" {
			// Resource.find consumes all_projects and Resource.list drops it.
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, list, Invalid("find query %q must be UTF-8 JSON", key)
		}
		items, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, list, err
		}
		if len(items) != 0 {
			member[key] = slices.Clone(items)
		}
		if wire, known := s.Attrs[key]; known {
			if previous, exists := seed[wire]; exists && !bytes.Equal(previous, raw) {
				return nil, nil, list, Invalid("find query aliases conflict for %q", wire)
			}
			seed[wire] = bytes.Clone(raw)
		}
		list.Query[key] = bytes.Clone(raw)
	}
	if _, named := values["name"]; !named {
		raw, _ := json.Marshal(nameOrID)
		list.Query["name"] = raw
	}
	return member, seed, list, nil
}
