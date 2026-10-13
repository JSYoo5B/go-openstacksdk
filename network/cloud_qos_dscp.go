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

// DSCPMarkingCloudOpts owns JSON-domain Source filters. The list is fetched before
// search selection. Truthy nonmapping filters fail at Source **kwargs expansion.
type DSCPMarkingCloudOpts struct {
	Filters         *json.RawMessage
	CurrentLocation *resource.CloudLocation
}
type DSCPMarkingCloudOption func(*DSCPMarkingCloudOpts) error

func copyDSCPMarkingCloud(c DSCPMarkingCloudOpts) DSCPMarkingCloudOpts {
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
func WithDSCPMarkingCloudOpts(c DSCPMarkingCloudOpts) DSCPMarkingCloudOption {
	owned := copyDSCPMarkingCloud(c)
	return func(c *DSCPMarkingCloudOpts) error { *c = copyDSCPMarkingCloud(owned); return nil }
}
func WithDSCPMarkingCloudFiltersJSON(raw json.RawMessage) DSCPMarkingCloudOption {
	owned := bytes.Clone(raw)
	return func(c *DSCPMarkingCloudOpts) error {
		x := json.RawMessage(bytes.Clone(owned))
		c.Filters = &x
		return nil
	}
}

// DSCPMarkingCloudResult separates extension and policy evidence, complete/partial
// rule inventory, the final yielded rule receipt and a successful Value.
// A failed HTTP page keeps its actual body, headers and status in the error.
type DSCPMarkingCloudResult struct {
	Extensions []*extensions.ExtensionRecord
	Policy     *policies.PolicyRecord
	Rules      []*rules.DSCPMarkingRecord
	Inventory  []*rules.DSCPMarkingRecord
	LastRule   *rules.DSCPMarkingRecord
	Rule       *rules.DSCPMarkingRecord
	Values     []any
	Value      json.RawMessage
}

func DSCPMarkingCloudWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "qos rules", cloudread.ContextError(ctx, err))
}
func (s *Service) ListQoSDSCPMarkingRules(ctx context.Context, policyNameOrID string, options ...DSCPMarkingCloudOption) (*DSCPMarkingCloudResult, error) {
	return s.readQoSDSCPMarking(ctx, policyNameOrID, "", false, options)
}
func (s *Service) SearchQoSDSCPMarkingRules(ctx context.Context, policyNameOrID, ruleID string, options ...DSCPMarkingCloudOption) (*DSCPMarkingCloudResult, error) {
	return s.readQoSDSCPMarking(ctx, policyNameOrID, ruleID, true, options)
}
func (s *Service) readQoSDSCPMarking(ctx context.Context, policyNameOrID, ruleID string, search bool, options []DSCPMarkingCloudOption) (*DSCPMarkingCloudResult, error) {
	operation := "ListQoSDSCPMarkingRules"
	if search {
		operation = "SearchQoSDSCPMarkingRules"
	}
	state, err := qosReadStart(ctx, s, "qos")
	if state == nil {
		return nil, DSCPMarkingCloudWrap(ctx, operation, err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	result := &DSCPMarkingCloudResult{Extensions: slices.Clone(state.Extensions)}
	fail := func(err error) (*DSCPMarkingCloudResult, error) {
		result.Value = nil
		result.Rules = nil
		result.Values = nil
		return result, DSCPMarkingCloudWrap(ctx, operation, err)
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
	cfg := copyDSCPMarkingCloud(DSCPMarkingCloudOpts{})
	for _, option := range slices.Clone(options) {
		if err := state.Check(ctx); err != nil {
			return fail(err)
		}
		if option == nil {
			return fail(fmt.Errorf("%w: nil QoS cloud option", resource.ErrInvalidOption))
		}
		next := copyDSCPMarkingCloud(cfg)
		if err := errors.Join(option(&next), state.Check(ctx)); err != nil {
			return fail(err)
		}
		cfg = copyDSCPMarkingCloud(next)
	}
	var id string
	if err = json.Unmarshal(result.Policy.Resource.Body["id"], &id); err != nil {
		return fail(dscpMarkingCloudPolicyProof(result.Policy, fmt.Errorf("%w: policy identity must be string: %v", resource.ErrInvalidOption, err)))
	}
	var leaf []rules.DSCPMarkingListOption
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
				leaf = append(leaf, rules.WithDSCPMarkingProxyQuery(member.Key, json.RawMessage(bytes.Clone(member.Value))))
			}
		}
	}
	proxy, e := state.API.QoSRules.ProxyDSCPMarkingRecords(ctx, id, leaf...)
	if e = errors.Join(e, state.Check(ctx)); e != nil {
		return fail(e)
	}
	result.Values = make([]any, 0)
	if generator, ok := proxy.Value.(*rules.DSCPMarkingGenerator); ok {
		for record, e := range generator.Records {
			if record != nil {
				result.LastRule = dscpMarkingCloudRecord(record)
			}
			if e = errors.Join(e, state.Check(ctx)); e != nil {
				return fail(e)
			}
			if e := dscpMarkingCloudLocation(record, cfg.CurrentLocation); e != nil {
				return fail(dscpMarkingCloudProof(record, e))
			}
			result.Inventory = append(result.Inventory, dscpMarkingCloudRecord(record))
			result.Values = append(result.Values, dscpMarkingCloudRecord(record))
		}
	} else {
		result.Values, e = dscpMarkingCloudIterable(proxy.Value)
		if e != nil {
			return fail(e)
		}
		dscpMarkingCloudBindGenerators(result.Values, cfg.CurrentLocation, state, ctx)
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
				if generator, ok := value.(*rules.DSCPMarkingGenerator); ok {
					for record, e := range generator.Records {
						if record != nil {
							result.LastRule = dscpMarkingCloudRecord(record)
						}
						if e = errors.Join(e, state.Check(ctx)); e != nil {
							return fail(e)
						}
						result.Inventory = append(result.Inventory, dscpMarkingCloudRecord(record))
					}
					return fail(dscpMarkingCloudProof(result.LastRule, fmt.Errorf("Source filter key is not an attribute of an opaque generator")))
				}
				if _, actual := value.(*rules.DSCPMarkingRecord); !actual {
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
		if record, ok := value.(*rules.DSCPMarkingRecord); ok {
			rows[i], e = json.Marshal(record.Resource.Body)
		} else {
			rows[i], e = json.Marshal(value)
		}
		if e != nil {
			completeJSON = false
			if search {
				return fail(dscpMarkingCloudProof(result.LastRule, e))
			}
		}
	}

	if search {
		selected, e := cloudfilter.SelectLoaded(rows, ruleID, cfg.Filters, func() error { return state.Check(ctx) })
		if e != nil {
			return fail(dscpMarkingCloudProof(result.LastRule, e))
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
	result.Rules = make([]*rules.DSCPMarkingRecord, 0)
	for _, value := range result.Values {
		if record, ok := value.(*rules.DSCPMarkingRecord); ok {
			result.Rules = append(result.Rules, dscpMarkingCloudRecord(record))
		}
	}

	if err := state.Check(ctx); err != nil {
		result.Value = nil
		result.Rules = nil
		return fail(err)
	}
	return result, nil
}

// GetQoSDSCPMarkingRule uses the Source member getter after one extension gate and
// policy lookup. It preserves strict member errors and never invokes Find/list
// fallback for a missing rule. The policy lookup policy is family-specific.
func (s *Service) GetQoSDSCPMarkingRule(ctx context.Context, policyNameOrID, ruleID string, options ...DSCPMarkingCloudOption) (*DSCPMarkingCloudResult, error) {
	const operation = "GetQoSDSCPMarkingRule"
	state, err := qosReadStart(ctx, s, "qos")
	if state == nil {
		return nil, DSCPMarkingCloudWrap(ctx, operation, err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	result := &DSCPMarkingCloudResult{Extensions: slices.Clone(state.Extensions)}
	fail := func(err error) (*DSCPMarkingCloudResult, error) {
		result.Value = nil
		result.Rules = nil
		result.Values = nil
		return result, DSCPMarkingCloudWrap(ctx, operation, err)
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
	cfg := copyDSCPMarkingCloud(DSCPMarkingCloudOpts{})
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return fail(fmt.Errorf("%w: nil cloud option", resource.ErrInvalidOption))
		}
		next := copyDSCPMarkingCloud(cfg)
		if e := errors.Join(apply(&next), state.Check(ctx)); e != nil {
			return fail(e)
		}
		cfg = copyDSCPMarkingCloud(next)
	}
	if cfg.Filters != nil {
		return fail(fmt.Errorf("%w: Source member getter has no filters", resource.ErrInvalidOption))
	}

	var id string
	if err = json.Unmarshal(result.Policy.Resource.Body["id"], &id); err != nil {
		return fail(dscpMarkingCloudPolicyProof(result.Policy, fmt.Errorf("%w: policy identity must be string: %v", resource.ErrInvalidOption, err)))
	}
	result.LastRule, err = state.API.QoSRules.GetSourceDSCPMarkingRecord(ctx, id, ruleID)
	if err != nil {
		return fail(err)
	}
	if err = state.Check(ctx); err != nil {
		return fail(err)
	}
	if e := dscpMarkingCloudLocation(result.LastRule, cfg.CurrentLocation); e != nil {
		return fail(dscpMarkingCloudProof(result.LastRule, e))
	}
	result.Rule = dscpMarkingCloudRecord(result.LastRule)
	result.Inventory = []*rules.DSCPMarkingRecord{dscpMarkingCloudRecord(result.LastRule)}
	if result.Rule != nil {
		result.Value, _ = json.Marshal(result.Rule.Resource.Body)
	}
	return result, nil
}

// WithDSCPMarkingCloudCurrentLocation snapshots connected Source location. Standalone
// APIs keep the pinned unconnected Resource location (null).
func WithDSCPMarkingCloudCurrentLocation(value resource.CloudLocation) DSCPMarkingCloudOption {
	owned := value.Clone()
	return func(c *DSCPMarkingCloudOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}
func dscpMarkingCloudRecord(v *rules.DSCPMarkingRecord) *rules.DSCPMarkingRecord {
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
func dscpMarkingCloudProof(v *rules.DSCPMarkingRecord, err error) error {
	if v == nil {
		return err
	}
	return (&rest.Response{Body: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}).Fail(err)
}
func dscpMarkingCloudPolicyProof(v *policies.PolicyRecord, err error) error {
	if v == nil {
		return err
	}
	return (&rest.Response{Body: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}).Fail(err)
}
func dscpMarkingCloudLocation(v *rules.DSCPMarkingRecord, location *resource.CloudLocation) error {
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
func dscpMarkingCloudIterable(value any) ([]any, error) {
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
func dscpMarkingCloudBindGenerators(values []any, location *resource.CloudLocation, state *qosReadState, ctx context.Context) {
	visited := make(map[*rules.DSCPMarkingGenerator]bool)
	var bind func(any)
	bind = func(value any) {
		switch v := value.(type) {
		case *rules.DSCPMarkingGenerator:
			if visited[v] {
				return
			}
			visited[v] = true
			original := v.Records
			v.Records = func(yield func(*rules.DSCPMarkingRecord, error) bool) {
				for row, err := range original {
					if err == nil {
						err = dscpMarkingCloudLocation(row, location)
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
