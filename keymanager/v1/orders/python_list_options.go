package orders

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordListOpts owns Source kwargs and read controls. Pagination defaults to
// true, zero MaxItems is unbounded, and nil Microversion retains the selected
// client's default. An absent default causes no Barbican version discovery.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	MaxItems     int
	Paginated    *bool
}
type RecordListOption func(*RecordListOpts) error

func ownRecordListOpts(value RecordListOpts) RecordListOpts {
	value.Query = maps.Clone(value.Query)
	for key, raw := range value.Query {
		value.Query[key] = bytes.Clone(raw)
	}
	value.Headers = maps.Clone(value.Headers)
	if value.Microversion != nil {
		x := *value.Microversion
		value.Microversion = &x
	}
	if value.Paginated != nil {
		x := *value.Paginated
		value.Paginated = &x
	}
	return value
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := ownRecordListOpts(value)
	return func(c *RecordListOpts) error { *c = ownRecordListOpts(owned); return nil }
}

// WithRecordListQuery follows the finite Source query/Body declaration. Unknown
// kwargs are discarded, not promoted to wire extensions or caller predicates.
func WithRecordListQuery(key string, value any) RecordListOption {
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
func WithRecordListHeaders(values map[string]string) RecordListOption {
	owned := maps.Clone(values)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordListMicroversion(value string) RecordListOption {
	return func(c *RecordListOpts) error { x := value; c.Microversion = &x; return nil }
}
func WithRecordListMaxItems(value int) RecordListOption {
	return func(c *RecordListOpts) error { c.MaxItems = value; return nil }
}
func WithRecordListPaginated(value bool) RecordListOption {
	return func(c *RecordListOpts) error { x := value; c.Paginated = &x; return nil }
}
func recordListInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

var recordListServerQueries = map[string]string{"limit": "limit", "marker": "marker"}
var recordListBodyAttributes = map[string]bool{"created_at": true, "creator_id": true, "meta": true, "order_ref": true, "order_id": true, "secret_ref": true, "secret_id": true, "status": true, "sub_status": true, "sub_status_message": true, "type": true, "updated_at": true, "id": true, "name": true}

func prepareRecordListQuery(policy RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query := make(url.Values)
	filters := make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: policy.MaxItems, SinglePage: policy.Paginated != nil && !*policy.Paginated, LimitHint: true}
	if policy.MaxItems < 0 {
		return nil, nil, control, recordListInvalid("max items must be nonnegative")
	}
	for key, raw := range policy.Query {
		if !utf8.ValidString(key) || strings.TrimSpace(key) == "" || strings.IndexFunc(key, unicode.IsControl) >= 0 {
			return nil, nil, control, recordListInvalid("query key must be nonblank UTF-8 without controls")
		}
		switch key {
		case "paginated", "max_items", "headers", "microversion", "base_path", "jmespath_filters", "__conflicting_attrs", "cls", "self":
			return nil, nil, control, recordListInvalid("query %q conflicts with the owned list policy", key)
		}
		// Unknown kwargs have no Source Body attribute and are discarded without
		// validating their JSON payload. Selected declared values are checked below.
		if _, server := recordListServerQueries[key]; !server && recordListBodyAttributes[key] {
			if !utf8.Valid(raw) || !json.Valid(raw) {
				return nil, nil, control, recordListInvalid("Body filter %q must be complete UTF-8 JSON", key)
			}
			filters[key] = bytes.Clone(raw)
		}
	}
	for client, wire := range recordListServerQueries {
		raw, present := policy.Query[client]
		if !present {
			raw, present = policy.Query[wire]
		}
		if !present {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, control, recordListInvalid("server query %q must be complete UTF-8 JSON", client)
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, control, err
		}
		if len(values) > 0 {
			query[wire] = slices.Clone(values)
		}
	}
	return query, filters, control, nil
}
func recordListLocalMatch(actual, filter json.RawMessage) (bool, error) {
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
			matched, err := recordListLocalMatch(observed[key], value)
			if err != nil || !matched {
				return matched, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}
