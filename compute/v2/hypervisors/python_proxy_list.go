package hypervisors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordGenerator preserves Source's unconsumed generator identity.
type RecordGenerator struct{ Records iter.Seq2[*ReadRecord, error] }

func (g *RecordGenerator) JMESPathIdentity() any { return g }
func (*RecordGenerator) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("%w: opaque hypervisor generator is not JSON", resource.ErrInvalidOption)
}

type ProxyRecordResult struct {
	Value any
	JSON  json.RawMessage
}

func recordJSON(v any) bool {
	switch x := v.(type) {
	case nil, bool, string, json.Number, float64:
		return true
	case []any:
		for _, c := range x {
			if !recordJSON(c) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, c := range x {
			if !recordJSON(c) {
				return false
			}
		}
		return true
	}
	return false
}

// ProxyListRecords evaluates JMESPath against the same opaque lazy generator
// as Source Proxy._list. It uses the shared JMESPath interpreter and identity ABI.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordOption) (*ProxyRecordResult, error) {
	p, err := a.prepareRecordPolicy(ctx, options, false)
	if err != nil {
		return nil, recordWrap(ctx, "ProxyListRecords", err)
	}
	policy, err := p.listPolicy(ctx)
	if err != nil {
		return nil, recordWrap(ctx, "ProxyListRecords", err)
	}
	generator := &RecordGenerator{Records: func(yield func(*ReadRecord, error) bool) {
		if p.version == "" {
			profile := microversions.NovaProfile
			profile.StopFaultRetryCallbacks = true
			if len(p.discovery.Responses) == 0 {
				p.discovery, err = microversions.Read(ctx, p.source, profile)
			}
			if err == nil {
				p.version, err = microversions.SelectVersion(p.discovery.Maximum, p.discovery.Minimum, "2.88")
			}
			if err != nil {
				yield(nil, recordWrap(ctx, "ProxyListRecords", err))
				return
			}
		}
		if _, explicit := p.options.Query["microversion"]; !explicit {
			policy.Microversion = &p.version
		}
		for v, cause := range p.engine.List(ctx, policy) {
			if !yield(p.record(v), recordWrap(ctx, "ProxyListRecords", cause)) {
				return
			}
		}
	}}
	var value any = generator
	var expression string
	if json.Unmarshal(p.expression, &expression) == nil && expression != "" {
		value, err = jmespath.Search(expression, generator)
	}
	if err = errors.Join(err, p.engine.Check(ctx)); err != nil {
		return nil, recordWrap(ctx, "ProxyListRecords", err)
	}
	result := &ProxyRecordResult{Value: value}
	if recordJSON(value) {
		result.JSON, err = json.Marshal(value)
	}
	return result, recordWrap(ctx, "ProxyListRecords", err)
}
