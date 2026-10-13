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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// QoSRuleTypeListOpts owns the original cloud filters JSON. Nil and JSON falsey
// values mean no kwargs; a truthy value must be an object. Transport/Resource
// controls remain subject to the owned rule type reader's explicit policy.
type QoSRuleTypeListOpts struct{ Filters json.RawMessage }
type QoSRuleTypeListOption func(*QoSRuleTypeListOpts) error

func cloneQoSRuleTypeListOpts(v QoSRuleTypeListOpts) QoSRuleTypeListOpts {
	v.Filters = bytes.Clone(v.Filters)
	return v
}
func WithQoSRuleTypeListOpts(v QoSRuleTypeListOpts) QoSRuleTypeListOption {
	owned := cloneQoSRuleTypeListOpts(v)
	return func(c *QoSRuleTypeListOpts) error { *c = cloneQoSRuleTypeListOpts(owned); return nil }
}
func WithQoSRuleTypeListFilters(raw json.RawMessage) QoSRuleTypeListOption {
	return WithQoSRuleTypeListOpts(QoSRuleTypeListOpts{Filters: raw})
}

// QoSRuleTypeListResult separates Source list() values from actual HTTP rows.
// RuleTypes contains records only when a real generator was collected; JSON
// literals never acquire Resource/Wire or HTTP evidence. On late errors only
// observed Inventory and Extensions remain, not a successful Values/JSON list.
type QoSRuleTypeListResult struct {
	Values     []any
	JSON       json.RawMessage
	RuleTypes  []*ruletypes.QoSRuleTypeRecord
	Inventory  []*ruletypes.QoSRuleTypeRecord
	Extensions []*extensions.ExtensionRecord
}

// ListQoSRuleTypeValues performs a fresh complete QoS extension gate before
// filters/options, then the pinned cloud list_qos_rule_types list() conversion.
// Stock plugin detail reads require admin and are deliberately not implemented
// by this core-user helper.
func (s *Service) ListQoSRuleTypeValues(ctx context.Context, options ...QoSRuleTypeListOption) (*QoSRuleTypeListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListQoSRuleTypeValues", "qos rule types", cloudread.ContextError(ctx, err))
	}
	state, err := qosReadStart(ctx, s, "qos")
	if err != nil {
		if state != nil {
			return &QoSRuleTypeListResult{Extensions: slices.Clone(state.Extensions), Inventory: make([]*ruletypes.QoSRuleTypeRecord, 0)}, wrap(err)
		}
		return nil, wrap(err)
	}
	result := &QoSRuleTypeListResult{Extensions: slices.Clone(state.Extensions), Inventory: make([]*ruletypes.QoSRuleTypeRecord, 0)}
	fail := func(err error) (*QoSRuleTypeListResult, error) {
		return result, wrap(qosRuleTypeListProof(result, err))
	}
	policy := QoSRuleTypeListOpts{}
	for _, apply := range slices.Clone(options) {
		if err := state.Check(ctx); err != nil {
			return fail(err)
		}
		if apply == nil {
			return fail(fmt.Errorf("%w: nil QoS rule type list option", resource.ErrInvalidOption))
		}
		next := cloneQoSRuleTypeListOpts(policy)
		err := errors.Join(apply(&next), state.Check(ctx))
		if err != nil {
			return fail(err)
		}
		policy = cloneQoSRuleTypeListOpts(next)
	}
	reader, err := qosRuleTypeListFilters(policy.Filters)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return fail(err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	proxy, err := state.API.QoSRuleTypes.ProxyListRecords(ctx, reader...)
	if err != nil {
		return fail(err)
	}
	value := proxy.Value
	if actual, ok := value.(*ruletypes.ProxyListGenerator); ok {
		values := make([]any, 0)
		records := make([]*ruletypes.QoSRuleTypeRecord, 0)
		for record, err := range actual.Records {
			if err != nil {
				return result, wrap(err)
			}
			result.Inventory = append(result.Inventory, record)
			values = append(values, record)
			records = append(records, record)
		}
		result.Values, result.RuleTypes = values, records
	} else {
		result.Values, err = qosRuleTypeListIterable(value)
		if err != nil {
			return fail(err)
		}
	}
	representations := make([]any, len(result.Values))
	complete := true
	for i, value := range result.Values {
		if err := state.Check(ctx); err != nil {
			result.Values, result.RuleTypes = nil, nil
			return fail(err)
		}
		if record, ok := value.(*ruletypes.QoSRuleTypeRecord); ok {
			representations[i] = record.Resource.Body
			continue
		}
		representations[i] = value
		if !qosRuleTypeListJSON(value) {
			complete = false
		}
	}
	if complete {
		result.JSON, err = json.Marshal(representations)
		if err != nil {
			result.Values, result.RuleTypes = nil, nil
			return fail(err)
		}
	}
	if err := state.Check(ctx); err != nil {
		result.Values, result.RuleTypes, result.JSON = nil, nil, nil
		return fail(err)
	}
	return result, nil
}

func qosRuleTypeListFilters(raw json.RawMessage) ([]ruletypes.ProxyListOption, error) {
	if raw == nil {
		return nil, nil
	}
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil || !truthy {
		return nil, err
	}
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: truthy rule type filters must be a kwargs object: %v", resource.ErrInvalidOption, err)
	}
	options := make([]ruletypes.ProxyListOption, 0, len(members))
	for _, member := range members {
		options = append(options, ruletypes.WithProxyListQuery(member.Key, member.Value))
	}
	return options, nil
}
func qosRuleTypeListIterable(value any) ([]any, error) {
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
		for _, character := range v {
			result = append(result, string(character))
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%w: Source list() cannot iterate %T", resource.ErrInvalidOption, value)
	}
}
func qosRuleTypeListJSON(value any) bool {
	switch v := value.(type) {
	case nil, bool, string, json.Number, float64:
		return true
	case []any:
		for _, child := range v {
			if !qosRuleTypeListJSON(child) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, child := range v {
			if !qosRuleTypeListJSON(child) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
func qosRuleTypeListProof(result *QoSRuleTypeListResult, err error) error {
	if err == nil || result == nil {
		return err
	}
	if len(result.Inventory) > 0 {
		last := result.Inventory[len(result.Inventory)-1]
		return (&rest.Response{Body: last.Envelope, Header: last.Header, StatusCode: last.StatusCode}).Fail(err)
	}
	if len(result.Extensions) > 0 {
		last := result.Extensions[len(result.Extensions)-1]
		return (&rest.Response{Body: last.Envelope, Header: last.Header, StatusCode: last.StatusCode}).Fail(err)
	}
	return err
}
