package policies

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordListGenerator preserves the actual lazy Resource.list result used by
// the pinned Proxy._list. An expression can return this generator directly or
// include it inside another value without making an HTTP request.
type RecordListGenerator struct {
	Records iter.Seq2[*PolicyRecord, error]
}

// JMESPathIdentity exposes the same opaque object to the shared evaluator.
func (g *RecordListGenerator) JMESPathIdentity() any { return g }

// MarshalJSON rejects the opaque generator without consuming it. Python's
// to_string(@) also fails when asked to JSON-serialize Resource.list.
func (*RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, errors.New("a lazy QoS policy record generator is not JSON")
}

// RecordProxyListResult keeps the actual Proxy result. Value is either the
// opaque generator, arbitrary JSON, or a JSON-shaped value containing that
// same generator. JSON is populated only for completely serializable results.
// A null expression result has Value nil and JSON "null".
type RecordProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

func policyProxyHasGenerator(value any) bool {
	switch v := value.(type) {
	case *RecordListGenerator:
		return true
	case []any:
		for _, item := range v {
			if policyProxyHasGenerator(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range v {
			if policyProxyHasGenerator(item) {
				return true
			}
		}
	}
	return false
}

// ProxyListRecords preserves Proxy.qos_policies(**query), including the deprecated
// JMESPath branch. Resource.list is a generator, not a collected JSON array:
// projection/index/type expressions can return null without HTTP; @ returns
// the generator; length(@) can fail before the first request. Use ListRecords
// or AllRecords for the ordinary resource-only Go convenience.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ProxyListRecords", "policies", cloudread.ContextError(ctx, err))
	}
	var operationGuard func(context.Context) error
	engine, err := a.policyProxyListEngine(ctx, func(ctx context.Context) error {
		if operationGuard != nil {
			return operationGuard(ctx)
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	policy, expressionRaw, err := applyPolicyProxyListOptions(ctx, engine, slices.Clone(options))
	if err != nil {
		return nil, wrap(err)
	}
	operationGuard = policy.operationGuard
	if err := engine.Check(ctx); err != nil {
		return nil, wrap(err)
	}
	expression, err := policyProxyExpression(expressionRaw)
	if err != nil {
		return nil, wrap(err)
	}
	generator := &RecordListGenerator{Records: func(yield func(*PolicyRecord, error) bool) {
		for value, err := range engine.List(ctx, policyListOptions(policy)) {
			if !yield(policyRecord(value), wrap(err)) {
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
	if !policyProxyHasGenerator(result.Value) {
		result.JSON, err = json.Marshal(result.Value)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	return result, wrap(engine.Check(ctx))
}

// The Source Proxy lane opts into exact link/equality/fault policies without
// changing existing typed List/Get contracts.
func (a *API) policyProxyListEngine(ctx context.Context, extra func(context.Context) error) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, policyRecordInvalid("configured policies API is required")
	}
	selected := a.client
	profile := policyRecordProfile
	profile.Body = slices.Clone(profile.Body)
	profile.Query = slices.Clone(profile.Query)
	profile.SourceLinkPolicy, profile.PythonLoadedEquality, profile.StopFaultRetryCallbacks = true, true, true
	return neutronfind.New(ctx, selected, profile, func(ctx context.Context) error {
		if a.client != selected {
			return policyRecordInvalid("selected policies client changed")
		}
		if extra != nil {
			return extra(ctx)
		}
		return nil
	})
}

func applyPolicyProxyListOptions(ctx context.Context, engine *neutronfind.Engine, options []RecordListOption) (RecordListOpts, json.RawMessage, error) {
	callbacks := make([]func(*RecordListOpts) error, len(options))
	for i, option := range options {
		callbacks[i] = option
	}
	policy, err := applyPolicyRecordOptions(ctx, engine, RecordListOpts{}, copyPolicyList, callbacks)
	if err != nil {
		return policy, nil, err
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(policyListOptions(policy))
	if err != nil {
		return policy, expression, err
	}
	policy.BasePath, policy.Microversion, policy.Query, policy.Headers, policy.order = normalized.BasePath, normalized.Microversion, normalized.Query, normalized.Headers, normalized.Order
	return copyPolicyList(policy), expression, engine.Check(ctx)
}
func policyProxyExpression(raw json.RawMessage) (string, error) {
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return "", err
	}
	if !truthy {
		return "", nil
	}
	var expression string
	if json.Unmarshal(raw, &expression) != nil {
		return "", nil
	}
	return expression, nil
}
