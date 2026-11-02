package backups

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordFindOpts defaults IgnoreMissing and Details to true. Query, Headers,
// MaxItems and Paginated are explicit Go extensions beyond the pinned proxy's
// two options, exposing the underlying Backup Resource read policies.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Details       *bool
	Query         map[string]json.RawMessage
	Headers       map[string]string
	MaxItems      int
	Paginated     *bool
}

// RecordFindOption is one option of FindRecord: WithRecordFindOpts,
// WithRecordFindDetails, WithRecordFindMaxItems, WithRecordFindPaginated,
// request.Custom over *RecordFindOpts, request.IgnoreMissing, request.Headers
// (replaces the headers) or request.Query (a Resource.find keyword argument
// sent as JSON: member GET sees ordinary Requests query values; the fallback
// list classifies declared server filters and Body-local filters and drops
// unknown kwargs). Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordFindOption = request.FindCall

var recordFindCalls = calls.Spec[RecordFindOpts]{
	IgnoreMissing: calls.Pointer(func(c *RecordFindOpts) **bool { return &c.IgnoreMissing }),
	Headers:       calls.Value(func(c *RecordFindOpts) *map[string]string { return &c.Headers }),
	Query: calls.JSONQuery(func(c *RecordFindOpts, key string, raw json.RawMessage, err error) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = raw
		return nil
	}),
}

func copyFindOpts(value RecordFindOpts) RecordFindOpts {
	value.Query = findFields(value.Query)
	value.Headers = maps.Clone(value.Headers)
	if value.IgnoreMissing != nil {
		x := *value.IgnoreMissing
		value.IgnoreMissing = &x
	}
	if value.Details != nil {
		x := *value.Details
		value.Details = &x
	}
	if value.Paginated != nil {
		x := *value.Paginated
		value.Paginated = &x
	}
	return value
}
func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyFindOpts(value)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil })
}
func WithRecordFindDetails(value bool) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { x := value; c.Details = &x; return nil })
}
func WithRecordFindMaxItems(value int) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { c.MaxItems = value; return nil })
}
func WithRecordFindPaginated(value bool) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { x := value; c.Paginated = &x; return nil })
}

func findKey(key string) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return findInvalid("query key must be nonblank UTF-8")
	}
	for _, r := range key {
		if unicode.IsControl(r) {
			return findInvalid("query key has controls")
		}
	}
	if key == "id" || key == "base_path" || key == "microversion" || key == "details" || key == "ignore_missing" {
		return findInvalid("query %q conflicts with the owned lookup policy", key)
	}
	return nil
}

var findServerQueries = map[string]string{
	"limit": "limit", "marker": "marker", "offset": "offset", "project_id": "project_id", "name": "name", "status": "status", "volume_id": "volume_id", "sort_key": "sort_key", "sort_dir": "sort_dir", "sort": "sort", "all_projects": "all_tenants",
}

func findServerKnown(key string) bool {
	if _, ok := findServerQueries[key]; ok {
		return true
	}
	for _, wire := range findServerQueries {
		if key == wire {
			return true
		}
	}
	return false
}
func prepareFindQuery(identity string, policy RecordFindOpts) (url.Values, url.Values, map[string]json.RawMessage, map[string]json.RawMessage, rest.ListControl, error) {
	getQuery, listQuery := make(url.Values), make(url.Values)
	filters, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: policy.MaxItems, SinglePage: policy.Paginated != nil && !*policy.Paginated, LimitHint: true}
	if policy.MaxItems < 0 {
		return nil, nil, nil, nil, control, findInvalid("max items must be nonnegative")
	}
	for key, raw := range policy.Query {
		if err := findKey(key); err != nil {
			return nil, nil, nil, nil, control, err
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, nil, nil, control, findInvalid("query %q must be complete UTF-8 JSON", key)
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		// Resource.find binds all_projects separately, excluding it from GET.
		if key != "all_projects" && len(values) > 0 {
			getQuery[key] = slices.Clone(values)
		}
		if canonical, ok := findCanonical(key); ok {
			seed[canonical] = bytes.Clone(raw)
		}
		switch key {
		case "max_items":
			var amount int
			if err := json.Unmarshal(raw, &amount); err != nil || amount < 0 {
				return nil, nil, nil, nil, control, findInvalid("max_items must be a nonnegative Go integer")
			}
			control.MaxItems = amount
		case "paginated":
			var enabled bool
			if err := json.Unmarshal(raw, &enabled); err != nil {
				return nil, nil, nil, nil, control, findInvalid("paginated must be boolean")
			}
			control.SinglePage = !enabled
		default:
			// Body aliases are not class attributes in Source. Only canonical
			// attribute names can become local filters; wire aliases may seed GET.
			if findBodyKnown(key) && !findServerKnown(key) {
				filters[key] = bytes.Clone(raw)
			}
		}
	}
	for client, wire := range findServerQueries {
		raw, ok := policy.Query[client]
		if !ok {
			raw, ok = policy.Query[wire]
		}
		if !ok {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		if len(values) > 0 {
			listQuery[wire] = slices.Clone(values)
		}
	}
	if _, present := policy.Query["name"]; !present {
		listQuery.Set("name", identity)
	}
	return getQuery, listQuery, filters, seed, control, nil
}
func findLocalMatch(actual, filter json.RawMessage) (bool, error) {
	if len(actual) == 0 {
		actual = json.RawMessage("null")
	}
	trimmed := bytes.TrimSpace(filter)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var wanted, observed map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &wanted); err != nil {
			return false, err
		}
		if err := json.Unmarshal(actual, &observed); err != nil || len(observed) == 0 {
			return false, nil
		}
		for key, value := range wanted {
			matched, err := findLocalMatch(observed[key], value)
			if err != nil || !matched {
				return matched, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}
