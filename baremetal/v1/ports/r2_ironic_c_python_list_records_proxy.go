package ports

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordListGenerator preserves the actual lazy Resource.list result used by
// the pinned Proxy._list. An expression can return this generator directly or
// include it inside another value without making an HTTP request.
type RecordListGenerator struct {
	Records iter.Seq2[*PortRecord, error]
}

// JMESPathIdentity exposes the same opaque object to the shared evaluator.
func (g *RecordListGenerator) JMESPathIdentity() any { return g }

// MarshalJSON rejects the opaque generator without consuming it. Python's
// to_string(@) also fails when asked to JSON-serialize Resource.list.
func (*RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, errors.New("a lazy resource record generator is not JSON")
}

// RecordProxyListResult keeps the actual Proxy result. Value is either the
// opaque generator, arbitrary JSON, or a JSON-shaped value containing that
// same generator. JSON is populated only for completely serializable results.
// A null expression result has Value nil and JSON "null".
type RecordProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

func listRecordProxyHasGenerator(value any) bool {
	switch v := value.(type) {
	case *RecordListGenerator:
		return true
	case []any:
		for _, item := range v {
			if listRecordProxyHasGenerator(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range v {
			if listRecordProxyHasGenerator(item) {
				return true
			}
		}
	}
	return false
}

// ProxyListRecords preserves Proxy.ports(**query), including the deprecated
// JMESPath branch. Resource.list is a generator, not a collected JSON array:
// projection/index/type expressions can return null without HTTP; @ returns
// the generator; length(@) can fail before the first request. Use ListRecords
// or AllRecords for the ordinary resource-only Go convenience.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ProxyListRecords", "ports", cloudread.ContextError(ctx, err))
	}
	engine, err := a.listRecordRecordEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	client := portReadClientSnapshot(a)
	policy, err := applyRecordListOptions(ctx, engine, slices.Clone(options))
	if err != nil {
		return nil, wrap(err)
	}
	expression, err := recordListExpression(&policy)
	if err != nil {
		return nil, wrap(err)
	}
	generator := &RecordListGenerator{Records: func(yield func(*PortRecord, error) bool) {
		selected, discovery, err := a.versionedReadEngine(ctx, engine, policy.Microversion, client, policy.Headers)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		owned := copyRecordListOpts(policy)
		owned.Microversion = nil
		for value, err := range selected.List(ctx, recordListOptions(owned)) {
			row := listRecordListRecord(value)
			if row != nil {
				row.Discovery = clonePortDiscovery(discovery)
			}
			if !yield(row, wrap(err)) {
				return
			}
		}
	}}
	result := &RecordProxyListResult{Value: generator}
	if expression != "" {
		result.Value, err = jmespath.Search(expression, generator)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	if !listRecordProxyHasGenerator(result.Value) {
		result.JSON, err = json.Marshal(result.Value)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	return result, wrap(engine.Check(ctx))
}
