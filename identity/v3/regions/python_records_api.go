package regions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func locationPreflight(value *resource.CloudLocation) error {
	if value == nil {
		return nil
	}
	_, err := value.ForResource(nil, nil)
	return err
}
func recordLocation(value *RegionRecord, location *resource.CloudLocation) error {
	if value == nil || value.Resource == nil || location == nil {
		return nil
	}
	// Region declares no project/zone fields: explicit current connection facts
	// own location, and unrelated response extension values remain in Wire.
	raw, err := location.ForResource(nil, nil)
	if err != nil {
		return err
	}
	value.Resource.Body["location"] = bytes.Clone(raw)
	return nil
}

func (a *API) prepareRecordList(ctx context.Context, options []RecordListOption) (*neutronfind.Engine, RecordListOpts, error) {
	engine, err := a.recordEngine(ctx)
	if err != nil {
		return nil, RecordListOpts{}, err
	}
	policy, err := applyRegionOptions(ctx, engine, RecordListOpts{}, copyListOptions, options)
	if err == nil {
		policy, err = normalizeListOptions(policy)
	}
	if err == nil {
		err = locationPreflight(policy.Location)
	}
	return engine, policy, errors.Join(err, engine.Check(ctx))
}
func recordExpression(policy *RecordListOpts) (*string, error) {
	raw := policy.proxyExpression
	policy.proxyExpression = nil
	if raw == nil {
		return nil, nil
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, recordInvalid("jmespath_filters must be complete UTF-8 JSON")
	}
	truth, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, err
	}
	trim := bytes.TrimSpace(raw)
	if !truth || trim[0] != '"' {
		return nil, nil
	}
	var expression string
	if err := json.Unmarshal(trim, &expression); err != nil {
		return nil, err
	}
	return &expression, nil
}
func listRegionRecords(ctx context.Context, engine *neutronfind.Engine, policy RecordListOpts, operation string) iter.Seq2[*RegionRecord, error] {
	owned := copyListOptions(policy)
	return func(yield func(*RegionRecord, error) bool) {
		for value, err := range engine.List(ctx, listPolicy(owned)) {
			record := regionRecord(value)
			if err == nil {
				err = errors.Join(recordLocation(record, owned.Location), engine.Check(ctx))
			}
			if err != nil && record != nil {
				record.Resource = nil
			}
			if !yield(record, recordWrap(ctx, operation, err)) {
				return
			}
		}
	}
}

// ListRecords lazily reads Region records with Source query classification and
// raw-row caps. Polymorphic Proxy expressions use ProxyListRecords instead.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*RegionRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*RegionRecord, error) bool) {
		engine, policy, err := a.prepareRecordList(ctx, owned)
		if err != nil {
			yield(nil, recordWrap(ctx, "ListRecords", err))
			return
		}
		expression, err := recordExpression(&policy)
		if err == nil && expression != nil {
			err = recordInvalid("JMESPath changes the Proxy return type; use ProxyListRecords")
		}
		if err != nil {
			yield(nil, recordWrap(ctx, "ListRecords", err))
			return
		}
		for value, err := range listRegionRecords(ctx, engine, policy, "ListRecords") {
			if !yield(value, err) {
				return
			}
		}
	}
}
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*RegionRecord, error) {
	records := make([]*RegionRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		records = append(records, value)
	}
	return records, nil
}

// RecordListGenerator preserves the unconsumed Python Resource.list value.
// Records performs guarded HTTP only when iterated; it cannot be JSON encoded.
type RecordListGenerator struct {
	Records iter.Seq2[*RegionRecord, error]
}

func (g *RecordListGenerator) JMESPathIdentity() any { return g }
func (*RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, recordInvalid("a Python list generator is not JSON serializable")
}

// RecordProxyListResult can contain an opaque generator, nested generators,
// scalar/null, or other expression values. JSON is present only for JSON data.
type RecordProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

func proxyJSON(value any) bool {
	switch v := value.(type) {
	case nil, bool, string, json.Number, int, int64, float64:
		return true
	case []any:
		for _, child := range v {
			if !proxyJSON(child) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, child := range v {
			if !proxyJSON(child) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// ProxyListRecords mirrors Source Proxy.regions, including its opaque generator
// JMESPath input. Expressions do not materialize rows or trigger HTTP merely
// to return null or a type error. Cloud search explicitly materializes first.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	const op = "ProxyListRecords"
	engine, policy, err := a.prepareRecordList(ctx, slices.Clone(options))
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	expression, err := recordExpression(&policy)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	generator := &RecordListGenerator{Records: listRegionRecords(ctx, engine, policy, op)}
	var value any = generator
	if expression != nil {
		value, err = jmespath.Search(*expression, generator)
		if err != nil {
			err = errors.Join(recordInvalid("Source Proxy list expression"), err)
		}
	}
	if err = errors.Join(err, engine.Check(ctx)); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	result := &RecordProxyListResult{Value: value}
	if proxyJSON(value) {
		result.JSON, err = json.Marshal(value)
	}
	return result, recordWrap(ctx, op, errors.Join(err, engine.Check(ctx)))
}
