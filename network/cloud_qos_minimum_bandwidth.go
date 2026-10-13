package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// MinimumBandwidthCloudOpts owns JSON-domain Source filters. The list is fetched before
// search selection. Truthy nonmapping filters fail at Source **kwargs expansion.
type MinimumBandwidthCloudOpts struct {
	Filters         *json.RawMessage
	CurrentLocation *resource.CloudLocation
}
type MinimumBandwidthCloudOption func(*MinimumBandwidthCloudOpts) error

func copyMinimumBandwidthCloud(c MinimumBandwidthCloudOpts) MinimumBandwidthCloudOpts {
	if c.CurrentLocation != nil {
		v := c.CurrentLocation.Clone()
		c.CurrentLocation = &v
	}
	if c.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*c.Filters))
		c.Filters = &raw
	}
	return c
}
func WithMinimumBandwidthCloudOpts(c MinimumBandwidthCloudOpts) MinimumBandwidthCloudOption {
	owned := copyMinimumBandwidthCloud(c)
	return func(c *MinimumBandwidthCloudOpts) error { *c = copyMinimumBandwidthCloud(owned); return nil }
}
func WithMinimumBandwidthCloudFiltersJSON(raw json.RawMessage) MinimumBandwidthCloudOption {
	owned := bytes.Clone(raw)
	return func(c *MinimumBandwidthCloudOpts) error {
		x := json.RawMessage(bytes.Clone(owned))
		c.Filters = &x
		return nil
	}
}

// MinimumBandwidthCloudResult separates extension and policy evidence, complete/partial
// rule inventory, the final yielded rule receipt and a successful Value.
// A failed HTTP page keeps its actual body, headers and status in the error.
type MinimumBandwidthCloudResult struct {
	Extensions []*extensions.ExtensionRecord
	Policy     *policies.PolicyRecord
	Rules      []*rules.MinimumBandwidthRecord
	Inventory  []*rules.MinimumBandwidthRecord
	LastRule   *rules.MinimumBandwidthRecord
	Rule       *rules.MinimumBandwidthRecord
	Values     []any
	Value      json.RawMessage
}

func MinimumBandwidthCloudWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "qos rules", cloudread.ContextError(ctx, err))
}
func (s *Service) ListQoSMinimumBandwidthRules(ctx context.Context, policyNameOrID string, options ...MinimumBandwidthCloudOption) (*MinimumBandwidthCloudResult, error) {
	return s.readQoSMinimumBandwidth(ctx, policyNameOrID, "", false, options)
}
func (s *Service) SearchQoSMinimumBandwidthRules(ctx context.Context, policyNameOrID, ruleID string, options ...MinimumBandwidthCloudOption) (*MinimumBandwidthCloudResult, error) {
	return s.readQoSMinimumBandwidth(ctx, policyNameOrID, ruleID, true, options)
}
func (s *Service) readQoSMinimumBandwidth(ctx context.Context, policyNameOrID, ruleID string, search bool, options []MinimumBandwidthCloudOption) (*MinimumBandwidthCloudResult, error) {
	operation := "ListQoSMinimumBandwidthRules"
	if search {
		operation = "SearchQoSMinimumBandwidthRules"
	}
	state, err := qosReadStart(ctx, s, "qos")
	if state == nil {
		return nil, MinimumBandwidthCloudWrap(ctx, operation, err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	result := &MinimumBandwidthCloudResult{Extensions: slices.Clone(state.Extensions)}
	fail := func(err error) (*MinimumBandwidthCloudResult, error) {
		result.Value = nil
		result.Rules = nil
		result.Values = nil
		return result, MinimumBandwidthCloudWrap(ctx, operation, err)
	}
	if err != nil {
		return fail(err)
	}
	result.Policy, err = state.Policy(ctx, policyNameOrID)
	if err != nil {
		return fail(err)
	}
	if result.Policy == nil {
		return fail(&resource.NotFoundError{Resource: "QoS policy", Reference: policyNameOrID})
	}
	cfg := copyMinimumBandwidthCloud(MinimumBandwidthCloudOpts{})
	for _, option := range slices.Clone(options) {
		if err := state.Check(ctx); err != nil {
			return fail(err)
		}
		if option == nil {
			return fail(fmt.Errorf("%w: nil QoS cloud option", resource.ErrInvalidOption))
		}
		next := copyMinimumBandwidthCloud(cfg)
		if err := errors.Join(option(&next), state.Check(ctx)); err != nil {
			return fail(err)
		}
		cfg = copyMinimumBandwidthCloud(next)
	}
	var id string
	if err = json.Unmarshal(result.Policy.Resource.Body["id"], &id); err != nil {
		return fail(minimumBandwidthCloudPolicyProof(result.Policy, fmt.Errorf("%w: policy identity must be string: %v", resource.ErrInvalidOption, err)))
	}
	var leaf []rules.MinimumBandwidthListOption
	if cfg.Filters != nil {
		truthy, e := cloudfilter.PythonJSONTruthy(*cfg.Filters)
		if e != nil {
			return fail(e)
		}
		if truthy {
			members, e := cloudfilter.ObjectMembers(*cfg.Filters)
			if e != nil {
				return fail(fmt.Errorf("Source filters must be a kwargs mapping: %w", e))
			}
			for _, member := range members {
				leaf = append(leaf, rules.WithMinimumBandwidthProxyQuery(member.Key, json.RawMessage(bytes.Clone(member.Value))))
			}
		}
	}
	proxy, e := state.API.QoSRules.ProxyMinimumBandwidthRecords(ctx, id, leaf...)
	if e = errors.Join(e, state.Check(ctx)); e != nil {
		return fail(e)
	}
	result.Values = make([]any, 0)
	if generator, ok := proxy.Value.(*rules.MinimumBandwidthGenerator); ok {
		for record, e := range generator.Records {
			if record != nil {
				result.LastRule = minimumBandwidthCloudRecord(record)
			}
			if e = errors.Join(e, state.Check(ctx)); e != nil {
				return fail(e)
			}
			if e := minimumBandwidthCloudLocation(record, cfg.CurrentLocation); e != nil {
				return fail(minimumBandwidthCloudProof(record, e))
			}
			result.Inventory = append(result.Inventory, minimumBandwidthCloudRecord(record))
			result.Values = append(result.Values, minimumBandwidthCloudRecord(record))
		}
	} else {
		result.Values, e = minimumBandwidthCloudIterable(proxy.Value)
		if e != nil {
			return fail(e)
		}
		minimumBandwidthCloudBindGenerators(result.Values, cfg.CurrentLocation, state, ctx)
	}

	// Source's mapping pass first skips falsey literals. Identifier selection
	// calls get() before this skip. Opaque generator membership consumes the
	// generator before raising an unknown-key error, retaining actual receipts.
	if search && ruleID == "" && cfg.Filters != nil {
		truthy, e := cloudfilter.PythonJSONTruthy(*cfg.Filters)
		if e != nil {
			return fail(e)
		}
		if truthy {
			values := make([]any, 0, len(result.Values))
			for _, value := range result.Values {
				if generator, ok := value.(*rules.MinimumBandwidthGenerator); ok {
					for record, e := range generator.Records {
						if record != nil {
							result.LastRule = minimumBandwidthCloudRecord(record)
						}
						if e = errors.Join(e, state.Check(ctx)); e != nil {
							return fail(e)
						}
						result.Inventory = append(result.Inventory, minimumBandwidthCloudRecord(record))
					}
					return fail(minimumBandwidthCloudProof(result.LastRule, fmt.Errorf("Source filter key is not an attribute of an opaque generator")))
				}
				if _, actual := value.(*rules.MinimumBandwidthRecord); !actual {
					encoded, e := json.Marshal(value)
					if e != nil {
						return fail(e)
					}
					nonzero, e := cloudfilter.PythonJSONTruthy(encoded)
					if e != nil {
						return fail(e)
					}
					if !nonzero {
						continue
					}
				}
				values = append(values, value)
			}
			result.Values = values
		}
	}
	rows := make([]json.RawMessage, len(result.Values))
	completeJSON := true
	for i, value := range result.Values {
		var e error
		if record, ok := value.(*rules.MinimumBandwidthRecord); ok {
			rows[i], e = json.Marshal(record.Resource.Body)
		} else {
			rows[i], e = json.Marshal(value)
		}
		if e != nil {
			completeJSON = false
			if search {
				return fail(minimumBandwidthCloudProof(result.LastRule, e))
			}
		}
	}

	if search {
		selected, e := cloudfilter.SelectLoaded(rows, ruleID, cfg.Filters, func() error { return state.Check(ctx) })
		if e != nil {
			return fail(minimumBandwidthCloudProof(result.LastRule, e))
		}
		result.Value = bytes.Clone(selected.Value)
		values := make([]any, len(selected.Indices))
		for i, index := range selected.Indices {
			values[i] = result.Values[index]
		}
		result.Values = values
	} else {
		if completeJSON {
			result.Value, _ = json.Marshal(rows)
		}
	}
	result.Rules = make([]*rules.MinimumBandwidthRecord, 0)
	for _, value := range result.Values {
		if record, ok := value.(*rules.MinimumBandwidthRecord); ok {
			result.Rules = append(result.Rules, minimumBandwidthCloudRecord(record))
		}
	}

	if err := state.Check(ctx); err != nil {
		result.Value = nil
		result.Rules = nil
		return fail(err)
	}
	return result, nil
}

// GetQoSMinimumBandwidthRule uses the Source member getter after one extension gate and
// policy lookup. It preserves strict member errors and never invokes Find/list
// fallback for a missing rule. The policy lookup policy is family-specific.
func (s *Service) GetQoSMinimumBandwidthRule(ctx context.Context, policyNameOrID, ruleID string, options ...MinimumBandwidthCloudOption) (*MinimumBandwidthCloudResult, error) {
	const operation = "GetQoSMinimumBandwidthRule"
	state, err := qosReadStart(ctx, s, "qos")
	if state == nil {
		return nil, MinimumBandwidthCloudWrap(ctx, operation, err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	result := &MinimumBandwidthCloudResult{Extensions: slices.Clone(state.Extensions)}
	fail := func(err error) (*MinimumBandwidthCloudResult, error) {
		result.Value = nil
		result.Rules = nil
		result.Values = nil
		return result, MinimumBandwidthCloudWrap(ctx, operation, err)
	}
	if err != nil {
		return fail(err)
	}
	result.Policy, err = state.PolicyStrict(ctx, policyNameOrID)
	if err != nil {
		return fail(err)
	}
	if result.Policy == nil {
		return fail(&resource.NotFoundError{Resource: "QoS policy", Reference: policyNameOrID})
	}
	cfg := copyMinimumBandwidthCloud(MinimumBandwidthCloudOpts{})
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return fail(fmt.Errorf("%w: nil cloud option", resource.ErrInvalidOption))
		}
		next := copyMinimumBandwidthCloud(cfg)
		if e := errors.Join(apply(&next), state.Check(ctx)); e != nil {
			return fail(e)
		}
		cfg = copyMinimumBandwidthCloud(next)
	}
	if cfg.Filters != nil {
		return fail(fmt.Errorf("%w: Source member getter has no filters", resource.ErrInvalidOption))
	}

	var id string
	if err = json.Unmarshal(result.Policy.Resource.Body["id"], &id); err != nil {
		return fail(minimumBandwidthCloudPolicyProof(result.Policy, fmt.Errorf("%w: policy identity must be string: %v", resource.ErrInvalidOption, err)))
	}
	result.LastRule, err = state.API.QoSRules.GetSourceMinimumBandwidthRecord(ctx, id, ruleID)
	if err != nil {
		return fail(err)
	}
	if err = state.Check(ctx); err != nil {
		return fail(err)
	}
	if e := minimumBandwidthCloudLocation(result.LastRule, cfg.CurrentLocation); e != nil {
		return fail(minimumBandwidthCloudProof(result.LastRule, e))
	}
	result.Rule = minimumBandwidthCloudRecord(result.LastRule)
	result.Inventory = []*rules.MinimumBandwidthRecord{minimumBandwidthCloudRecord(result.LastRule)}
	if result.Rule != nil {
		result.Value, _ = json.Marshal(result.Rule.Resource.Body)
	}
	return result, nil
}

// WithMinimumBandwidthCloudCurrentLocation snapshots connected Source location. Standalone
// APIs keep the pinned unconnected Resource location (null).
func WithMinimumBandwidthCloudCurrentLocation(value resource.CloudLocation) MinimumBandwidthCloudOption {
	owned := value.Clone()
	return func(c *MinimumBandwidthCloudOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}
func minimumBandwidthCloudRecord(v *rules.MinimumBandwidthRecord) *rules.MinimumBandwidthRecord {
	if v == nil {
		return nil
	}
	c := *v
	c.Resource = v.Resource.Clone()
	c.Wire = v.Wire.Clone()
	c.Header = v.Header.Clone()
	c.Envelope = bytes.Clone(v.Envelope)
	return &c
}
func minimumBandwidthCloudProof(v *rules.MinimumBandwidthRecord, err error) error {
	if v == nil {
		return err
	}
	return (&rest.Response{Body: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}).Fail(err)
}
func minimumBandwidthCloudPolicyProof(v *policies.PolicyRecord, err error) error {
	if v == nil {
		return err
	}
	return (&rest.Response{Body: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}).Fail(err)
}
func minimumBandwidthCloudLocation(v *rules.MinimumBandwidthRecord, location *resource.CloudLocation) error {
	if v == nil || v.Resource == nil {
		return fmt.Errorf("%w: missing QoS Resource", resource.ErrInvalidOption)
	}
	// These Source classes declare neither project_id nor availability_zone.
	// Unknown wire fields do not trigger Resource._update_location.
	if location != nil {
		raw, err := json.Marshal(location.Clone())
		if err != nil {
			return err
		}
		v.Resource.Body["location"] = raw
	}
	return nil
}
func minimumBandwidthCloudIterable(value any) ([]any, error) {
	switch v := value.(type) {
	case []any:
		return slices.Clone(v), nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		values := make([]any, len(keys))
		for i, k := range keys {
			values[i] = k
		}
		return values, nil
	case string:
		values := make([]any, 0)
		for _, c := range v {
			values = append(values, string(c))
		}
		return values, nil
	default:
		return nil, fmt.Errorf("%w: Source list() cannot iterate %T", resource.ErrInvalidOption, value)
	}
}
func minimumBandwidthCloudBindGenerators(values []any, location *resource.CloudLocation, state *qosReadState, ctx context.Context) {
	visited := make(map[*rules.MinimumBandwidthGenerator]bool)
	var bind func(any)
	bind = func(value any) {
		switch v := value.(type) {
		case *rules.MinimumBandwidthGenerator:
			if visited[v] {
				return
			}
			visited[v] = true
			original := v.Records
			v.Records = func(yield func(*rules.MinimumBandwidthRecord, error) bool) {
				for row, err := range original {
					if err == nil {
						err = minimumBandwidthCloudLocation(row, location)
					}
					err = errors.Join(err, state.Check(ctx))
					if !yield(row, err) || err != nil {
						return
					}
				}
			}
		case []any:
			for _, child := range v {
				bind(child)
			}
		case map[string]any:
			for _, child := range v {
				bind(child)
			}
		}
	}
	for _, value := range values {
		bind(value)
	}
}
