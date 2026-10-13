package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SourceCloudListOpts owns cloud.list_ports' filters and Go transport facts.
// A falsey filter is equivalent to {}; a truthy object becomes Proxy kwargs.
type SourceCloudListOpts struct {
	Filters        *json.RawMessage
	Headers        map[string]string
	Microversion   *string
	CloudLocation  *resource.CloudLocation
	headersPresent bool
}
type SourceCloudListOption func(*SourceCloudListOpts) error

func copySourceCloudListOpts(value SourceCloudListOpts) SourceCloudListOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	if value.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*value.Filters))
		value.Filters = &raw
	}
	if value.Microversion != nil {
		v := *value.Microversion
		value.Microversion = &v
	}
	if value.CloudLocation != nil {
		v := value.CloudLocation.Clone()
		value.CloudLocation = &v
	}
	return value
}
func WithSourceCloudListOpts(value SourceCloudListOpts) SourceCloudListOption {
	value.headersPresent = value.headersPresent || value.Headers != nil
	owned := copySourceCloudListOpts(value)
	return func(c *SourceCloudListOpts) error { *c = copySourceCloudListOpts(owned); return nil }
}
func WithSourceCloudListFilters(value json.RawMessage) SourceCloudListOption {
	owned := bytes.Clone(value)
	return func(c *SourceCloudListOpts) error {
		c.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Filters = &raw
		}
		return nil
	}
}
func WithSourceCloudListHeaders(value map[string]string) SourceCloudListOption {
	owned := maps.Clone(value)
	return func(c *SourceCloudListOpts) error { c.Headers = maps.Clone(owned); c.headersPresent = true; return nil }
}
func WithSourceCloudListMicroversion(value string) SourceCloudListOption {
	return func(c *SourceCloudListOpts) error { v := value; c.Microversion = &v; return nil }
}
func WithSourceCloudListLocation(value resource.CloudLocation) SourceCloudListOption {
	owned := value.Clone()
	return func(c *SourceCloudListOpts) error { v := owned.Clone(); c.CloudLocation = &v; return nil }
}

// SourceCloudListResult is the actual Python list() conversion. Values can
// contain JSON values, original PortRecord pointers or nested opaque generators.
// JSON is present only for a complete JSON-representable list. Ports is supplied
// only for an actual Resource generator; Inventory contains actual HTTP rows.
type SourceCloudListResult struct {
	Values    []any
	JSON      json.RawMessage
	Ports     []*PortRecord
	Inventory []*PortRecord
}

// ListSourceValues provides cloud.list_ports' polymorphic list conversion.
// It never normalizes a literal JSON mapping into a Port or fabricates a Wire,
// response body, Header or status for evidence-free values.
func (a *API) ListSourceValues(ctx context.Context, options ...SourceCloudListOption) (*SourceCloudListResult, error) {
	const op = "ListSourceValues"
	wrap := func(err error) error { return request.Wrap(op, "ports", cloudread.ContextError(ctx, err)) }
	engine, err := a.portListEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	cfg := copySourceCloudListOpts(SourceCloudListOpts{})
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil cloud port list option"))
		}
		next := copySourceCloudListOpts(cfg)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copySourceCloudListOpts(next)
	}
	pushdown, err := sourceCloudListPushdown(cfg.Filters)
	if err != nil {
		return nil, wrap(err)
	}
	if cfg.headersPresent || len(cfg.Headers) > 0 {
		pushdown = append(pushdown, WithRecordListHeaders(cfg.Headers))
	}
	if cfg.Microversion != nil {
		pushdown = append(pushdown, WithRecordListMicroversion(*cfg.Microversion))
	}
	policy, err := applyRecordListOptions(ctx, engine, pushdown)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := portProxyListValue(ctx, engine, policy, op)
	if err != nil {
		return nil, wrap(err)
	}
	result := &SourceCloudListResult{Inventory: make([]*PortRecord, 0)}
	if generator, ok := value.(*RecordListGenerator); ok {
		values := make([]any, 0)
		records := make([]*PortRecord, 0)
		for record, err := range generator.Records {
			if err != nil {
				return result, wrap(err)
			}
			result.Inventory = append(result.Inventory, record)
			if err := sourceCloudPortLocation(record, cfg.CloudLocation); err != nil {
				return result, wrap(portSubnetProof(record, err))
			}
			values = append(values, record)
			records = append(records, record)
		}
		result.Values, result.Ports = values, records
	} else {
		result.Values, err = sourceCloudIterable(value)
		if err != nil {
			return result, wrap(err)
		}
		sourceCloudWrapGenerators(result.Values, cfg.CloudLocation, engine, ctx)
	}
	representations := make([]any, len(result.Values))
	completeJSON := true
	for i, value := range result.Values {
		if err := engine.Check(ctx); err != nil {
			return result, wrap(err)
		}
		if record, ok := value.(*PortRecord); ok {
			representations[i] = record.Resource.Body
			continue
		}
		representations[i] = value
		if !recordProxyJSONValue(value) {
			completeJSON = false
		}
	}
	if completeJSON {
		result.JSON, err = json.Marshal(representations)
		if err != nil {
			return result, wrap(err)
		}
	}
	return result, wrap(engine.Check(ctx))
}

func sourceCloudListPushdown(filters *json.RawMessage) ([]RecordListOption, error) {
	if filters == nil {
		return nil, nil
	}
	truthy, err := cloudfilter.PythonJSONTruthy(*filters)
	if err != nil {
		return nil, err
	}
	if !truthy {
		return nil, nil
	}
	members, err := cloudfilter.ObjectMembers(*filters)
	if err != nil {
		return nil, errors.Join(findInvalid("truthy cloud list filters must be a mapping"), err)
	}
	options := make([]RecordListOption, 0, len(members))
	for _, member := range members {
		options = append(options, withRecordListRaw(member.Key, member.Value, nil))
	}
	return options, nil
}

func sourceCloudIterable(value any) ([]any, error) {
	switch v := value.(type) {
	case []any:
		return slices.Clone(v), nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		result := make([]any, len(keys))
		for i, key := range keys {
			result[i] = key
		}
		return result, nil
	case string:
		result := make([]any, 0)
		for _, char := range v {
			result = append(result, string(char))
		}
		return result, nil
	default:
		return nil, findInvalid("Source list() cannot iterate %T", value)
	}
}

func sourceCloudPortLocation(record *PortRecord, location *resource.CloudLocation) error {
	if location == nil {
		return nil
	}
	actual, err := location.ForResource(record.Resource.Body["project_id"], record.Resource.Body["availability_zone"])
	if err != nil {
		return err
	}
	record.Resource.Body["location"] = actual
	return nil
}

func sourceCloudWrapGenerators(values []any, location *resource.CloudLocation, engine *neutronfind.Engine, ctx context.Context) {
	visited := make(map[*RecordListGenerator]bool)
	var wrap func(any)
	wrap = func(value any) {
		switch v := value.(type) {
		case *RecordListGenerator:
			if visited[v] {
				return
			}
			visited[v] = true
			original := v.Records
			v.Records = func(yield func(*PortRecord, error) bool) {
				for record, err := range original {
					if err == nil {
						err = sourceCloudPortLocation(record, location)
					}
					err = errors.Join(err, engine.Check(ctx))
					if err != nil {
						yield(nil, request.Wrap("ListSourceValues", "ports", cloudread.ContextError(ctx, err)))
						return
					}
					if !yield(record, nil) {
						return
					}
				}
			}
		case []any:
			for _, child := range v {
				wrap(child)
			}
		case map[string]any:
			for _, child := range v {
				wrap(child)
			}
		}
	}
	for _, value := range values {
		wrap(value)
	}
}
