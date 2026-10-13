package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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

// QoSBandwidthLimitResult separates selected values from real rule, parent and
// extension records. Local literal values never acquire a fabricated receipt.
type QoSBandwidthLimitResult struct {
	Rules      []*rules.BandwidthLimitRecord
	Rule       *rules.BandwidthLimitRecord
	Value      json.RawMessage
	Values     []any
	Inventory  []*rules.BandwidthLimitRecord
	Policy     *policies.PolicyRecord
	Extensions []*extensions.ExtensionRecord
	filters    *json.RawMessage
}

type QoSBandwidthLimitOpts struct {
	Filters        *json.RawMessage
	Headers        map[string]string
	Microversion   *string
	headersPresent bool
}
type QoSBandwidthLimitOption func(*QoSBandwidthLimitOpts) error

func copyQoSBandwidthOpts(v QoSBandwidthLimitOpts) QoSBandwidthLimitOpts {
	if v.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*v.Filters))
		v.Filters = &raw
	}
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	if v.Microversion != nil {
		value := *v.Microversion
		v.Microversion = &value
	}
	return v
}
func WithQoSBandwidthLimitOpts(v QoSBandwidthLimitOpts) QoSBandwidthLimitOption {
	v.headersPresent = v.headersPresent || v.Headers != nil
	owned := copyQoSBandwidthOpts(v)
	return func(c *QoSBandwidthLimitOpts) error { *c = copyQoSBandwidthOpts(owned); return nil }
}
func WithQoSBandwidthLimitFilters(raw json.RawMessage) QoSBandwidthLimitOption {
	owned := bytes.Clone(raw)
	return func(c *QoSBandwidthLimitOpts) error {
		value := json.RawMessage(bytes.Clone(owned))
		c.Filters = &value
		return nil
	}
}
func WithQoSBandwidthLimitHeaders(value map[string]string) QoSBandwidthLimitOption {
	owned := maps.Clone(value)
	return func(c *QoSBandwidthLimitOpts) error {
		c.Headers = maps.Clone(owned)
		c.headersPresent = true
		return nil
	}
}
func WithQoSBandwidthLimitMicroversion(value string) QoSBandwidthLimitOption {
	return func(c *QoSBandwidthLimitOpts) error { v := value; c.Microversion = &v; return nil }
}
func qosBandwidthInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func qosBandwidthWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "qos_bandwidth_limit_rules", cloudread.ContextError(ctx, err))
}

func (s *Service) qosBandwidthStart(ctx context.Context, policy string, options []QoSBandwidthLimitOption) (*QoSBandwidthLimitResult, *qosReadState, QoSBandwidthLimitOpts, error) {
	result := &QoSBandwidthLimitResult{}
	state, err := qosReadStart(ctx, s, "qos")
	if state != nil {
		result.Extensions = slices.Clone(state.Extensions)
	}
	if err != nil {
		return result, state, QoSBandwidthLimitOpts{}, err
	}
	result.Policy, err = state.Policy(ctx, policy)
	if err != nil {
		return result, state, QoSBandwidthLimitOpts{}, err
	}
	if result.Policy == nil || result.Policy.Resource == nil {
		return result, state, QoSBandwidthLimitOpts{}, &resource.NotFoundError{Resource: "qos_policy", Reference: policy}
	}
	cfg := copyQoSBandwidthOpts(QoSBandwidthLimitOpts{})
	for _, apply := range slices.Clone(options) {
		if err := state.Check(ctx); err != nil {
			return result, state, cfg, err
		}
		if apply == nil {
			return result, state, cfg, qosBandwidthInvalid("nil bandwidth cloud option")
		}
		next := copyQoSBandwidthOpts(cfg)
		if err := errors.Join(apply(&next), state.Check(ctx)); err != nil {
			return result, state, cfg, err
		}
		cfg = copyQoSBandwidthOpts(next)
	}
	return result, state, cfg, state.Check(ctx)
}
func qosBandwidthParent(result *QoSBandwidthLimitResult) (string, error) {
	var id string
	if err := json.Unmarshal(result.Policy.Resource.Body["id"], &id); err != nil || id == "" {
		cause := errors.Join(qosBandwidthInvalid("resolved QoS policy requires string ID"), err)
		return "", (&rest.Response{Body: bytes.Clone(result.Policy.Envelope), Header: result.Policy.Header.Clone(), StatusCode: result.Policy.StatusCode}).Fail(cause)
	}
	return id, nil
}
func qosBandwidthListOptions(cfg QoSBandwidthLimitOpts) ([]rules.BandwidthLimitListOption, error) {
	options := make([]rules.BandwidthLimitListOption, 0)
	if cfg.Filters != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(*cfg.Filters)
		if err != nil {
			return nil, qosBandwidthInvalid("invalid filters: %v", err)
		}
		if truthy {
			members, err := cloudfilter.ObjectMembers(*cfg.Filters)
			if err != nil {
				return nil, qosBandwidthInvalid("truthy list filters must be a kwargs object: %v", err)
			}
			for _, member := range members {
				options = append(options, rules.WithBandwidthLimitProxyQuery(member.Key, json.RawMessage(bytes.Clone(member.Value))))
			}
		}
	}
	if cfg.headersPresent || len(cfg.Headers) > 0 {
		options = append(options, rules.WithBandwidthLimitListHeaders(cfg.Headers))
	}
	if cfg.Microversion != nil {
		options = append(options, rules.WithBandwidthLimitListMicroversion(*cfg.Microversion))
	}
	return options, nil
}

// QoSBandwidthLimitGenerator exposes the original proxy generator's identity.
type QoSBandwidthLimitGenerator = rules.BandwidthLimitGenerator

func qosBandwidthIterable(value any) ([]any, error) {
	switch v := value.(type) {
	case []any:
		return slices.Clone(v), nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		out := make([]any, len(keys))
		for i, key := range keys {
			out[i] = key
		}
		return out, nil
	case string:
		out := make([]any, 0)
		for _, char := range v {
			out = append(out, string(char))
		}
		return out, nil
	default:
		return nil, qosBandwidthInvalid("Source list result is not iterable")
	}
}
func qosBandwidthJSON(values []any) (json.RawMessage, error) {
	views := make([]any, len(values))
	for i, value := range values {
		if row, ok := value.(*rules.BandwidthLimitRecord); ok {
			views[i] = row.Resource.Body
		} else {
			views[i] = value
		}
	}
	return json.Marshal(views)
}
func qosBandwidthProof(result *QoSBandwidthLimitResult, cause error) error {
	if len(result.Inventory) == 0 {
		return cause
	}
	row := result.Inventory[len(result.Inventory)-1]
	return (&rest.Response{Body: bytes.Clone(row.Envelope), Header: row.Header.Clone(), StatusCode: row.StatusCode}).Fail(cause)
}

// ListQoSBandwidthLimitRules fully checks QoS extensions, resolves the parent
// with ignore_missing=true, then collects Source list() with owned kwargs.
func (s *Service) ListQoSBandwidthLimitRules(ctx context.Context, policy string, options ...QoSBandwidthLimitOption) (*QoSBandwidthLimitResult, error) {
	result, _, err := s.qosBandwidthList(ctx, policy, options)
	return result, qosBandwidthWrap(ctx, "ListQoSBandwidthLimitRules", err)
}
func (s *Service) qosBandwidthList(ctx context.Context, policy string, options []QoSBandwidthLimitOption) (*QoSBandwidthLimitResult, *qosReadState, error) {
	result, state, cfg, err := s.qosBandwidthStart(ctx, policy, options)
	if err != nil {
		return result, state, err
	}
	parent, err := qosBandwidthParent(result)
	if err != nil {
		return result, state, err
	}
	listOptions, err := qosBandwidthListOptions(cfg)
	if err != nil {
		return result, state, err
	}
	if cfg.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*cfg.Filters))
		result.filters = &raw
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	proxy, err := state.API.QoSRules.ProxyBandwidthLimitRecords(ctx, parent, listOptions...)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return result, state, err
	}
	value := proxy.Value
	values := make([]any, 0)
	if direct, ok := value.(*QoSBandwidthLimitGenerator); ok {
		result.Inventory = make([]*rules.BandwidthLimitRecord, 0)
		selected := make([]*rules.BandwidthLimitRecord, 0)
		for row, err := range direct.Records {
			if err = errors.Join(err, state.Check(ctx)); err != nil {
				return result, state, err
			}
			result.Inventory = append(result.Inventory, row)
			selected = append(selected, row)
			values = append(values, row)
		}
		result.Rules = selected
	} else {
		values, err = qosBandwidthIterable(value)
		if err != nil {
			return result, state, err
		}
	}
	if err := state.Check(ctx); err != nil {
		return result, state, err
	}
	result.Values = values
	result.Value, err = qosBandwidthJSON(values)
	if err != nil {
		result.Value = nil
		err = nil
	} // Opaque nested generators stay Values only.
	return result, state, err
}

// SearchQoSBandwidthLimitRules delegates the complete list workflow, then
// applies Source ID/glob/name and local dictionary filtering to normalized views.
func (s *Service) SearchQoSBandwidthLimitRules(ctx context.Context, policy, ruleID string, options ...QoSBandwidthLimitOption) (*QoSBandwidthLimitResult, error) {
	const op = "SearchQoSBandwidthLimitRules"
	result, state, err := s.qosBandwidthList(ctx, policy, options)
	if err != nil {
		return result, qosBandwidthWrap(ctx, op, err)
	}
	originalValues := result.Values
	result.Value, result.Values, result.Rules = nil, nil, nil
	var filters *json.RawMessage
	// The list owns callback results. Reusing callbacks would run user code twice.
	// Snapshotting the applied filter is carried through the private list result.
	if result.filters != nil {
		raw := json.RawMessage(bytes.Clone(*result.filters))
		filters = &raw
	}
	// Source skips falsey entries before dictionary operations only when the
	// identifier phase is absent. Membership on an opaque generator consumes
	// actual records before the first unknown-key failure.
	filteredValues := make([]any, 0, len(originalValues))
	filterTruthy := false
	if filters != nil {
		filterTruthy, err = cloudfilter.PythonJSONTruthy(*filters)
		if err != nil {
			return result, qosBandwidthWrap(ctx, op, err)
		}
	}
	for _, value := range originalValues {
		if ruleID == "" && filterTruthy {
			if generator, ok := value.(*QoSBandwidthLimitGenerator); ok {
				for row, failure := range generator.Records {
					if failure = errors.Join(failure, state.Check(ctx)); failure != nil {
						return result, qosBandwidthWrap(ctx, op, failure)
					}
					result.Inventory = append(result.Inventory, row)
				}
				return result, qosBandwidthWrap(ctx, op, qosBandwidthProof(result, qosBandwidthInvalid("filter key is not an attribute of an opaque generator")))
			}
			if _, actual := value.(*rules.BandwidthLimitRecord); !actual {
				encoded, failure := json.Marshal(value)
				if failure != nil {
					return result, qosBandwidthWrap(ctx, op, failure)
				}
				truthy, failure := cloudfilter.PythonJSONTruthy(encoded)
				if failure != nil {
					return result, qosBandwidthWrap(ctx, op, failure)
				}
				if !truthy {
					continue
				}
			}
		}
		filteredValues = append(filteredValues, value)
	}
	originalValues = filteredValues
	rows := make([]json.RawMessage, len(originalValues))
	for i, value := range originalValues {
		if err := state.Check(ctx); err != nil {
			return result, qosBandwidthWrap(ctx, op, err)
		}
		if row, ok := value.(*rules.BandwidthLimitRecord); ok {
			rows[i], err = json.Marshal(row.Resource.Body)
		} else {
			rows[i], err = json.Marshal(value)
		}
		if err != nil {
			return result, qosBandwidthWrap(ctx, op, qosBandwidthProof(result, err))
		}
	}
	selected, err := cloudfilter.SelectLoaded(rows, ruleID, filters, func() error { return state.Check(ctx) })
	if err != nil {
		return result, qosBandwidthWrap(ctx, op, qosBandwidthProof(result, err))
	}
	result.Value = bytes.Clone(selected.Value)
	result.Rules = nil
	if !selected.Expression {
		result.Rules = make([]*rules.BandwidthLimitRecord, 0, len(selected.Indices))
		values := make([]any, 0, len(selected.Indices))
		for _, index := range selected.Indices {
			value := originalValues[index]
			values = append(values, value)
			if row, ok := value.(*rules.BandwidthLimitRecord); ok {
				result.Rules = append(result.Rules, row)
			}
		}
		result.Values = values
	} else {
		result.Values = nil
	}
	return result, qosBandwidthWrap(ctx, op, state.Check(ctx))
}

// GetQoSBandwidthLimitRule always gates extensions and resolves policy first,
// then performs literal rule GET without rule Find fallback or ignored 404.
func (s *Service) GetQoSBandwidthLimitRule(ctx context.Context, policy, ruleID string, options ...QoSBandwidthLimitOption) (*QoSBandwidthLimitResult, error) {
	const op = "GetQoSBandwidthLimitRule"
	result, state, cfg, err := s.qosBandwidthStart(ctx, policy, options)
	if err != nil {
		return result, qosBandwidthWrap(ctx, op, err)
	}
	parent, err := qosBandwidthParent(result)
	if err != nil {
		return result, qosBandwidthWrap(ctx, op, err)
	}
	opts := []rules.BandwidthLimitGetOption{rules.WithBandwidthLimitGetHeaders(cfg.Headers)}
	if cfg.Microversion != nil {
		opts = append(opts, rules.WithBandwidthLimitGetMicroversion(*cfg.Microversion))
	}
	if cfg.Filters != nil {
		return result, qosBandwidthWrap(ctx, op, qosBandwidthInvalid("get has no filters argument"))
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	result.Rule, err = state.API.QoSRules.GetBandwidthLimitRecord(ctx, parent, ruleID, opts...)
	if result.Rule != nil {
		result.Inventory = []*rules.BandwidthLimitRecord{result.Rule}
	}
	err = errors.Join(err, state.Check(ctx))
	if err != nil {
		return result, qosBandwidthWrap(ctx, op, err)
	}
	result.Rule.Resource.Body["location"] = json.RawMessage("null")
	result.Value, err = json.Marshal(result.Rule.Resource.Body)
	return result, qosBandwidthWrap(ctx, op, err)
}
