package resourcefilters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordListGenerator is Source's still-unconsumed Resource.list value. The
// shared JMESPath OpaqueIdentity ABI preserves identity without serializing it.
type RecordListGenerator struct {
	Records iter.Seq2[*ResourceFilterRecord, error]
}

func (g *RecordListGenerator) JMESPathIdentity() any { return g }
func (*RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, invalid("a Python resource-filter generator is not JSON serializable")
}

// RecordProxyListResult preserves arbitrary JSON, an opaque generator, or a
// container holding the same generator pointer. JSON is set only for JSON.
type RecordProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

func proxyExpression(cfg RecordListOpts) (*string, error) {
	var raw json.RawMessage
	if value, exists := cfg.Query["jmespath_filters"]; exists {
		var err error
		raw, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	} else if cfg.JMESPath != nil {
		raw, _ = json.Marshal(*cfg.JMESPath)
	}
	if raw == nil {
		return nil, nil
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, invalid("jmespath_filters needs complete UTF-8 JSON")
	}
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, err
	}
	token := bytes.TrimSpace(raw)
	if !truthy || token[0] != '"' {
		return nil, nil
	}
	var expression string
	if err = json.Unmarshal(token, &expression); err != nil {
		return nil, err
	}
	return &expression, nil
}
func proxyJSON(value any) bool {
	switch value := value.(type) {
	case nil, bool, string, json.Number, float64:
		return true
	case []any:
		for _, child := range value {
			if !proxyJSON(child) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, child := range value {
			if !proxyJSON(child) {
				return false
			}
		}
		return true
	}
	return false
}

// ProxyListRecords mirrors the pinned Proxy resource_filters return type.
// JMESPath receives an opaque unconsumed generator before any discovery/HTTP:
// projections may return null, array functions fail, and @ retains the pointer.
// ListResult is the explicit eager alternative for evaluating actual rows.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ProxyListRecords", "resource_filters", cloudread.ContextError(ctx, err))
	}
	p, err := a.prepare(ctx, copyCallbacks(options), true, true)
	if err != nil {
		return nil, wrap(err)
	}
	generator := &RecordListGenerator{Records: func(yield func(*ResourceFilterRecord, error) bool) {
		for value, err := range p.stream() {
			if !yield(value, wrap(err)) {
				return
			}
		}
	}}
	var value any = generator
	if p.cfg.JMESPath != nil {
		value, err = jmespath.Search(*p.cfg.JMESPath, generator)
		if err = errors.Join(err, p.guard(ctx)); err != nil {
			return nil, wrap(errors.Join(invalid("Source Proxy generator expression"), err))
		}
	}
	result := &RecordProxyListResult{Value: value}
	if proxyJSON(value) {
		result.JSON, err = json.Marshal(value)
		if err != nil {
			return nil, wrap(err)
		}
	}
	return result, wrap(p.guard(ctx))
}
