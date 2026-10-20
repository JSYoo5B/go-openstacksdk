package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordSearchOpts holds the cloud search_ports filters: a JSON object for
// nested dictionary filters, or a JSON string for the deprecated JMESPath
// expression. Headers and Microversion apply to every list page.
type RecordSearchOpts struct {
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	// Location supplies the library-owned Connection view before local search.
	Location      *json.RawMessage
	CloudLocation *resource.CloudLocation
}
type RecordSearchOption func(*RecordSearchOpts) error

func copyRecordSearchOpts(v RecordSearchOpts) RecordSearchOpts {
	owned := copyRecordListOpts(RecordListOpts{Headers: v.Headers, Microversion: v.Microversion})
	v.Headers, v.Microversion = owned.Headers, owned.Microversion
	if v.CloudLocation != nil {
		copy := v.CloudLocation.Clone()
		v.CloudLocation = &copy
	}
	if v.Location != nil {
		raw := json.RawMessage(bytes.Clone(*v.Location))
		v.Location = &raw
	}
	if v.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*v.Filters))
		v.Filters = &raw
	}
	return v
}

// WithRecordSearchOpts snapshots the complete concrete policy, including raw
// filters and an optional Connection location. No user-supplied builder is needed.
func WithRecordSearchOpts(value RecordSearchOpts) RecordSearchOption {
	owned := copyRecordSearchOpts(value)
	return func(c *RecordSearchOpts) error { *c = copyRecordSearchOpts(owned); return nil }
}

// WithRecordSearchLocation is the Connection adapter's declared Resource view.
// It is applied before dictionary/JMESPath selection and never changes Wire.
func WithRecordSearchLocation(value json.RawMessage) RecordSearchOption {
	owned := bytes.Clone(value)
	return func(c *RecordSearchOpts) error {
		c.Location = nil
		c.CloudLocation = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Location = &raw
		}
		return nil
	}
}

// WithRecordSearchCloudLocation supplies configured Connection facts. The SDK
// computes each actual port's project/zone view before local selection, retaining
// foreign project IDs while clearing the current project's names.
func WithRecordSearchCloudLocation(value resource.CloudLocation) RecordSearchOption {
	owned := value.Clone()
	return func(c *RecordSearchOpts) error {
		copy := owned.Clone()
		c.CloudLocation = &copy
		c.Location = nil
		return nil
	}
}

// WithRecordSearchFilters snapshots raw JSON filters. nil clears them.
func WithRecordSearchFilters(value json.RawMessage) RecordSearchOption {
	owned := bytes.Clone(value)
	return func(c *RecordSearchOpts) error {
		c.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Filters = &raw
		}
		return nil
	}
}

// WithRecordSearchExpression supplies the deprecated JMESPath string filter.
// Python then lists every port without pushing any filter to Neutron.
func WithRecordSearchExpression(expression string) RecordSearchOption {
	raw, _ := json.Marshal(expression)
	return WithRecordSearchFilters(raw)
}
func WithRecordSearchHeaders(v map[string]string) RecordSearchOption {
	owned := copyRecordSearchOpts(RecordSearchOpts{Headers: v})
	return func(c *RecordSearchOpts) error {
		c.Headers = copyRecordSearchOpts(owned).Headers
		return nil
	}
}
func WithRecordSearchMicroversion(v string) RecordSearchOption {
	return func(c *RecordSearchOpts) error { x := v; c.Microversion = &x; return nil }
}

// SearchResult is a completed cloud search. Value is the selected JSON array
// of Python views, or the arbitrary JMESPath result. Ports keeps the selected
// records in order and is nil for an expression. Inventory is every listed
// record, including the rows read before a failure.
type SearchResult struct {
	Value     json.RawMessage
	Ports     []*PortRecord
	Inventory []*PortRecord
}

// searchPushdown mirrors search_ports/list_ports: a dictionary becomes
// Proxy.ports kwargs in insertion order, while a string or falsey value lists
// every port. Any other truthy JSON cannot be expanded as Python **kwargs.
func searchPushdown(filters *json.RawMessage) ([]RecordListOption, error) {
	if filters == nil {
		return nil, nil
	}
	raw := bytes.TrimSpace(*filters)
	if len(raw) == 0 || !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, findInvalid("search filters must contain complete JSON")
	}
	switch raw[0] {
	case '"':
		return nil, nil
	case '{':
		members, err := cloudfilter.ObjectMembers(raw)
		if err != nil {
			return nil, err
		}
		options := make([]RecordListOption, 0, len(members))
		for _, member := range members {
			options = append(options, withRecordListRaw(member.Key, member.Value, nil))
		}
		return options, nil
	}
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, err
	}
	if truthy {
		return nil, findInvalid("search filters must be a JSON object or a JMESPath string")
	}
	return nil, nil
}

// SearchRecords implements the cloud layer search_ports(name_or_id, filters):
// list_ports with dictionary filters pushed down as Proxy.ports kwargs (no
// pushdown for a JMESPath string), then _utils._filter_list with exact or
// glob ID/name matching and nested dictionary or JMESPath filters on the
// Python attribute views (Resource.Body).
func (a *API) SearchRecords(ctx context.Context, nameOrID string, options ...RecordSearchOption) (*SearchResult, error) {
	const op = "SearchRecords"
	wrap := func(err error) error { return request.Wrap(op, "ports", cloudread.ContextError(ctx, err)) }
	engine, err := a.portListEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy := copyRecordSearchOpts(RecordSearchOpts{})
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil ports search option"))
		}
		next := copyRecordSearchOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyRecordSearchOpts(next)
	}
	if policy.CloudLocation != nil {
		if _, err := policy.CloudLocation.ForResource(nil, nil); err != nil {
			return nil, wrap(err)
		}
	}
	if policy.Location != nil {
		raw := bytes.TrimSpace(*policy.Location)
		if !utf8.Valid(raw) || !json.Valid(raw) || !(bytes.Equal(raw, []byte("null")) || len(raw) > 0 && raw[0] == '{') {
			return nil, wrap(findInvalid("Connection location must be a JSON object or null"))
		}
	}
	pushdown, err := searchPushdown(policy.Filters)
	if err != nil {
		return nil, wrap(err)
	}
	if policy.Headers != nil {
		pushdown = append(pushdown, WithRecordListHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		pushdown = append(pushdown, WithRecordListMicroversion(*policy.Microversion))
	}
	listPolicy, err := applyRecordListOptions(ctx, engine, pushdown)
	if err != nil {
		return nil, wrap(err)
	}
	result := &SearchResult{Inventory: make([]*PortRecord, 0)}
	rows, records, err := collectPortSearchRows(ctx, engine, listPolicy, nameOrID, policy.Location, policy.CloudLocation, result)
	if err != nil {
		return result, wrap(err)
	}
	localFilters := policy.Filters
	if localFilters != nil {
		truthy, cause := cloudfilter.PythonJSONTruthy(*localFilters)
		if cause != nil {
			return result, wrap(cause)
		}
		if !truthy {
			localFilters = nil
		}
	}
	selected, err := cloudfilter.SelectLoaded(rows, nameOrID, localFilters, func() error { return engine.Check(ctx) })
	if err != nil {
		return result, wrap(fmt.Errorf("%w: local port search: %w", resource.ErrInvalidOption, err))
	}
	if !selected.Expression {
		result.Ports = make([]*PortRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			if records[index] == nil {
				result.Ports = nil
				break
			}
			result.Ports[i] = records[index]
		}
	}
	result.Value = bytes.Clone(selected.Value)
	return result, wrap(engine.Check(ctx))
}

// collectPortSearchRows preserves list_ports' polymorphic Proxy result. Literal
// JSON rows have no HTTP receipt or invented Port association; actual records
// retain independent evidence in Inventory. Non-iterable Source values fail
// before any HTTP, and a returned generator remains lazy until collection.
func collectPortSearchRows(ctx context.Context, engine *neutronfind.Engine, policy RecordListOpts, nameOrID string, location *json.RawMessage, cloudLocation *resource.CloudLocation, result *SearchResult) ([]json.RawMessage, []*PortRecord, error) {
	value, err := portProxyListValue(ctx, engine, policy, "SearchRecords")
	if err != nil {
		return nil, nil, err
	}
	rows := make([]json.RawMessage, 0)
	records := make([]*PortRecord, 0)
	if generator, ok := value.(*RecordListGenerator); ok {
		for record, err := range generator.Records {
			if err != nil {
				return nil, nil, err
			}
			if cloudLocation != nil {
				raw, err := cloudLocation.ForResource(record.Resource.Body["project_id"], record.Resource.Body["availability_zone"])
				if err != nil {
					return nil, nil, err
				}
				record.Resource.Body["location"] = raw
			} else if location != nil {
				record.Resource.Body["location"] = bytes.Clone(*location)
			}
			row, err := json.Marshal(record.Resource.Body)
			if err != nil {
				return nil, nil, err
			}
			result.Inventory = append(result.Inventory, record)
			rows = append(rows, row)
			records = append(records, record)
		}
		return rows, records, nil
	}
	var values []any
	switch value := value.(type) {
	case []any:
		values = value
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			values = append(values, key)
		}
	case string:
		for _, char := range value {
			values = append(values, string(char))
		}
	default:
		return nil, nil, findInvalid("list_ports cannot iterate the Source Proxy result %T", value)
	}
	for _, value := range values {
		if err := engine.Check(ctx); err != nil {
			return nil, nil, err
		}
		if !recordProxyJSONValue(value) {
			return nil, nil, findInvalid("cloud search row is an opaque Source value")
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, nil, err
		}
		if nameOrID == "" {
			truthy, cause := cloudfilter.PythonJSONTruthy(raw)
			if cause != nil {
				return nil, nil, cause
			}
			if !truthy {
				continue
			}
		}
		rows = append(rows, raw)
		records = append(records, nil)
	}
	return rows, records, nil
}
