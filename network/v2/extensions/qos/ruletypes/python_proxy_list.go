package ruletypes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

// ProxyListOpts is a separate Source kwargs lane. Existing RecordListOpts keeps
// its reviewed raw-reserved rejection policy. Order owns kwargs insertion order.
type ProxyListOpts struct {
	Query map[string]json.RawMessage
	Order []string
}
type ProxyListOption func(*ProxyListOpts) error

func cloneProxyListOpts(v ProxyListOpts) ProxyListOpts {
	owned := neutronfind.CloneOptions(neutronfind.Options{Query: v.Query, Order: v.Order})
	return ProxyListOpts{Query: owned.Query, Order: owned.Order}
}
func WithProxyListOpts(v ProxyListOpts) ProxyListOption {
	owned := cloneProxyListOpts(v)
	return func(c *ProxyListOpts) error { *c = cloneProxyListOpts(owned); return nil }
}
func WithProxyListQuery(key string, value any) ProxyListOption {
	if key != "jmespath_filters" && ruleTypeListIgnores(key) {
		return func(*ProxyListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *ProxyListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.Order, key) {
			c.Order = append(c.Order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}

type ProxyListGenerator struct {
	Records iter.Seq2[*QoSRuleTypeRecord, error]
}

func (g *ProxyListGenerator) JMESPathIdentity() any { return g }
func (*ProxyListGenerator) MarshalJSON() ([]byte, error) {
	return nil, ruleTypeRecordInvalid("Source rule type generator is not JSON serializable")
}

type ProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

// ProxyListRecords preserves the pinned qos_rule_types kwargs and generator
// JMESPath branch. Raw transport formal validation is deferred until iteration,
// so a literal expression does not validate unused headers or send a list GET.
func (a *API) ProxyListRecords(ctx context.Context, options ...ProxyListOption) (*ProxyListResult, error) {
	const op = "ProxyListRecords"
	wrap := func(err error) error { return ruleTypeRecordWrap(ctx, op, err) }
	engine, err := a.ruleTypeRecordEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	callbacks := make([]func(*ProxyListOpts) error, len(options))
	for i, option := range slices.Clone(options) {
		callbacks[i] = option
	}
	cfg, err := applyRuleTypeRecordOptions(ctx, engine, ProxyListOpts{}, cloneProxyListOpts, callbacks)
	if err != nil {
		return nil, wrap(err)
	}
	policy, pending, expression, err := prepareProxyRuleTypeList(cfg)
	if err = errors.Join(err, engine.Check(ctx)); err != nil {
		return nil, wrap(err)
	}
	generator := &ProxyListGenerator{Records: func(yield func(*QoSRuleTypeRecord, error) bool) {
		if err := engine.Check(ctx); err != nil {
			yield(nil, wrap(err))
			return
		}
		formal, _, err := neutronfind.NormalizeProxyListOptions(pending)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		owned := neutronfind.CloneOptions(policy)
		owned.BasePath, owned.Headers, owned.Microversion = formal.BasePath, formal.Headers, formal.Microversion
		for record, err := range engine.List(ctx, owned) {
			if !yield(ruleTypeRecord(record), wrap(err)) {
				return
			}
		}
	}}
	var value any = generator
	if expression != "" {
		value, err = jmespath.Search(expression, generator)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	result := &ProxyListResult{Value: value}
	if proxyRuleTypeJSON(value) {
		result.JSON, err = json.Marshal(value)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	return result, wrap(engine.Check(ctx))
}

// Only the three Resource.list formals have lazy value validation. Their names
// are still present while the common normalizer owns argument collisions and
// one ordered __conflicting_attrs pass. Null placeholders defer values without
// copying the resource-specific Body/query classifier or Source merge logic.
func prepareProxyRuleTypeList(cfg ProxyListOpts) (neutronfind.Options, neutronfind.Options, string, error) {
	policy := neutronfind.CloneOptions(neutronfind.Options{Query: cfg.Query, Order: cfg.Order})
	pending := neutronfind.Options{Query: make(map[string]json.RawMessage)}
	if policy.Query == nil {
		policy.Query = make(map[string]json.RawMessage)
	}
	for _, key := range []string{"base_path", "headers", "microversion"} {
		if raw, ok := policy.Query[key]; ok {
			pending.Query[key] = bytes.Clone(raw)
			pending.Order = append(pending.Order, key)
			policy.Query[key] = json.RawMessage("null")
		}
	}
	if raw, ok := policy.Query["__conflicting_attrs"]; ok {
		truthy, err := cloudfilter.PythonJSONTruthy(raw)
		if err != nil {
			return policy, pending, "", err
		}
		if truthy {
			members, err := cloudfilter.ObjectMembers(raw)
			if err != nil {
				return policy, pending, "", err
			}
			var sanitized bytes.Buffer
			sanitized.WriteByte('{')
			for i, member := range members {
				if i > 0 {
					sanitized.WriteByte(',')
				}
				key, _ := json.Marshal(member.Key)
				sanitized.Write(key)
				sanitized.WriteByte(':')
				if member.Key == "headers" || member.Key == "microversion" {
					if _, ok := pending.Query[member.Key]; !ok {
						pending.Order = append(pending.Order, member.Key)
					}
					pending.Query[member.Key] = bytes.Clone(member.Value)
					sanitized.WriteString("null")
				} else {
					sanitized.Write(member.Value)
				}
			}
			sanitized.WriteByte('}')
			policy.Query["__conflicting_attrs"] = bytes.Clone(sanitized.Bytes())
		}
	}
	policy, rawExpression, err := neutronfind.NormalizeProxyListOptions(policy)
	if err != nil {
		return policy, pending, "", err
	}
	truthy, err := cloudfilter.PythonJSONTruthy(rawExpression)
	if err != nil || !truthy {
		return policy, pending, "", err
	}
	trimmed := bytes.TrimSpace(rawExpression)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return policy, pending, "", nil
	}
	var expression string
	err = json.Unmarshal(rawExpression, &expression)
	return policy, pending, expression, err
}
func proxyRuleTypeJSON(value any) bool {
	switch v := value.(type) {
	case nil, bool, string, json.Number, float64:
		return true
	case []any:
		for _, child := range v {
			if !proxyRuleTypeJSON(child) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, child := range v {
			if !proxyRuleTypeJSON(child) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
