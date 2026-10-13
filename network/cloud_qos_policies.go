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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// QoSPolicyReadOpts owns filters passed to the Python cloud list/search phase.
// A nil Filters value is omitted; every JSON falsey value means an empty query.
// Search accepts a JSON object or the JSON-domain iterable-of-pairs accepted
// by dict.update. List requires an object for truthy filters.
type QoSPolicyReadOpts struct {
	Filters      json.RawMessage
	Headers      map[string]string
	Microversion *string
}
type QoSPolicyReadOption func(*QoSPolicyReadOpts) error

func copyQoSPolicyReadOpts(value QoSPolicyReadOpts) QoSPolicyReadOpts {
	value.Filters = bytes.Clone(value.Filters)
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	if value.Microversion != nil {
		v := *value.Microversion
		value.Microversion = &v
	}
	return value
}
func WithQoSPolicyReadOpts(value QoSPolicyReadOpts) QoSPolicyReadOption {
	owned := copyQoSPolicyReadOpts(value)
	return func(current *QoSPolicyReadOpts) error { *current = copyQoSPolicyReadOpts(owned); return nil }
}
func WithQoSPolicyFilters(value json.RawMessage) QoSPolicyReadOption {
	owned := bytes.Clone(value)
	return func(current *QoSPolicyReadOpts) error { current.Filters = bytes.Clone(owned); return nil }
}
func WithQoSPolicyHeaders(value map[string]string) QoSPolicyReadOption {
	owned := maps.Clone(value)
	return func(current *QoSPolicyReadOpts) error { current.Headers = maps.Clone(owned); return nil }
}
func WithQoSPolicyMicroversion(value string) QoSPolicyReadOption {
	return func(current *QoSPolicyReadOpts) error { owned := value; current.Microversion = &owned; return nil }
}

// QoSPolicyRecords retains both the completed extension gate and policy reads.
// Ordinary results have Policies and Value entries containing the corresponding
// Resource snapshots. Proxy expressions may instead return arbitrary list items
// or nested opaque generators. JSON is set only for completely serializable
// values. On error, Policies/Value/JSON are absent; actual completed extension
// evidence remains available and the error retains the failed HTTP receipt.
type QoSPolicyRecords struct {
	Policies   []*policies.PolicyRecord
	Extensions []*extensions.ExtensionRecord
	Value      []any
	JSON       json.RawMessage
}

func qosPolicyCloudInvalid(format string, values ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, values...))
}
func qosPolicyCloudWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "qos policies", cloudread.ContextError(ctx, err))
}

// ListQoSPolicyRecords implements NetworkCloudMixin.list_qos_policies: complete
// uncached qos extension discovery, falsey filters -> {}, then list(Proxy).
func (s *Service) ListQoSPolicyRecords(ctx context.Context, options ...QoSPolicyReadOption) (*QoSPolicyRecords, error) {
	return s.qosPolicyCloudRecords(ctx, "ListQoSPolicyRecords", "", false, options)
}

// SearchQoSPolicyRecords sends a truthy identity as a literal server name query,
// then lets filters.update replace it. It does not invent ID/glob/local search.
func (s *Service) SearchQoSPolicyRecords(ctx context.Context, nameOrID string, options ...QoSPolicyReadOption) (*QoSPolicyRecords, error) {
	return s.qosPolicyCloudRecords(ctx, "SearchQoSPolicyRecords", nameOrID, true, options)
}

func qosPolicyCloudPairs(raw json.RawMessage, search bool) ([]cloudfilter.JSONMember, error) {
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, qosPolicyCloudInvalid("filters must contain complete UTF-8 JSON: %v", err)
	}
	if !truthy {
		return nil, nil
	}
	token := bytes.TrimSpace(raw)
	if token[0] == '{' {
		return cloudfilter.ObjectMembers(raw)
	}
	if !search {
		return nil, qosPolicyCloudInvalid("truthy list filters must be a mapping for **filters")
	}
	if token[0] != '[' {
		return nil, qosPolicyCloudInvalid("search filters must be a mapping or iterable of key/value pairs")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(token, &entries); err != nil {
		return nil, err
	}
	members := make([]cloudfilter.JSONMember, 0, len(entries))
	for index, entry := range entries {
		var pair []json.RawMessage
		trimmed := bytes.TrimSpace(entry)
		if len(trimmed) > 0 && trimmed[0] == '"' {
			var text string
			if err := json.Unmarshal(entry, &text); err != nil {
				return nil, err
			}
			for _, char := range text {
				encoded, _ := json.Marshal(string(char))
				pair = append(pair, encoded)
			}
		} else if len(trimmed) > 0 && trimmed[0] == '{' {
			keys, err := cloudfilter.ObjectMembers(entry)
			if err != nil {
				return nil, err
			}
			for _, member := range keys {
				encoded, _ := json.Marshal(member.Key)
				pair = append(pair, encoded)
			}
		} else if err := json.Unmarshal(entry, &pair); err != nil {
			return nil, qosPolicyCloudInvalid("search update element %d is not a pair", index)
		}
		if len(pair) != 2 {
			return nil, qosPolicyCloudInvalid("search update element %d has length %d, expected 2", index, len(pair))
		}
		var key string
		if json.Unmarshal(pair[0], &key) != nil {
			return nil, qosPolicyCloudInvalid("search kwargs keys must be strings")
		}
		members = append(members, cloudfilter.JSONMember{Key: key, Value: bytes.Clone(pair[1])})
	}
	return members, nil
}

func qosPolicyCloudContainsOpaque(value any) bool {
	switch typed := value.(type) {
	case *policies.RecordListGenerator:
		return true
	case []any:
		for _, child := range typed {
			if qosPolicyCloudContainsOpaque(child) {
				return true
			}
		}
	case map[string]any:
		for _, child := range typed {
			if qosPolicyCloudContainsOpaque(child) {
				return true
			}
		}
	}
	return false
}

func (s *Service) qosPolicyCloudRecords(ctx context.Context, op, identity string, search bool, options []QoSPolicyReadOption) (*QoSPolicyRecords, error) {
	state, err := qosReadStart(ctx, s, "qos")
	if err != nil {
		var result *QoSPolicyRecords
		if state != nil {
			result = &QoSPolicyRecords{Extensions: state.Extensions}
		}
		return result, qosPolicyCloudWrap(ctx, op, err)
	}
	result := &QoSPolicyRecords{Extensions: state.Extensions}
	fail := func(err error) (*QoSPolicyRecords, error) { return result, qosPolicyCloudWrap(ctx, op, err) }
	policy := copyQoSPolicyReadOpts(QoSPolicyReadOpts{})
	for _, apply := range slices.Clone(options) {
		if err := state.Check(ctx); err != nil {
			return fail(err)
		}
		if apply == nil {
			return fail(qosPolicyCloudInvalid("nil QoS policy option"))
		}
		next := copyQoSPolicyReadOpts(policy)
		err := apply(&next)
		if err = errors.Join(err, state.Check(ctx)); err != nil {
			return fail(err)
		}
		policy = copyQoSPolicyReadOpts(next)
	}
	members, err := qosPolicyCloudPairs(policy.Filters, search)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return fail(err)
	}
	// Preserve dict.update insertion order and replacement at the original slot.
	ordered := make([]cloudfilter.JSONMember, 0, len(members)+1)
	if search && identity != "" {
		raw, _ := json.Marshal(identity)
		ordered = append(ordered, cloudfilter.JSONMember{Key: "name", Value: raw})
	}
	positions := make(map[string]int)
	for i, member := range ordered {
		positions[member.Key] = i
	}
	for _, member := range members {
		if i, exists := positions[member.Key]; exists {
			ordered[i].Value = bytes.Clone(member.Value)
		} else {
			positions[member.Key] = len(ordered)
			ordered = append(ordered, member)
		}
	}
	leaf := make([]policies.RecordListOption, 0, len(ordered)+3)
	for _, member := range ordered {
		leaf = append(leaf, policies.WithRecordListQuery(member.Key, member.Value))
	}
	// These concrete Go conveniences apply after the Source raw filter kwargs.
	if len(policy.Headers) > 0 {
		leaf = append(leaf, policies.WithRecordListHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		leaf = append(leaf, policies.WithRecordListMicroversion(*policy.Microversion))
	}
	leaf = append(leaf, policies.WithOperationGuard(state.Check))
	if err := state.Check(ctx); err != nil {
		return fail(err)
	}
	proxy, err := state.API.QoSPolicies.ProxyListRecords(ctx, leaf...)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return fail(err)
	}
	values := make([]any, 0)
	records := make([]*policies.PolicyRecord, 0)
	switch value := proxy.Value.(type) {
	case *policies.RecordListGenerator:
		for row, cause := range value.Records {
			if cause = errors.Join(cause, state.Check(ctx)); cause != nil {
				return fail(cause)
			}
			records = append(records, row)
			values = append(values, row.Resource)
		}
	case []any:
		values = append(values, value...)
	case string:
		for _, char := range value {
			values = append(values, string(char))
		}
	case map[string]any:
		// Go's shared JMES evaluator has no insertion-order metadata. This reviewed
		// Go difference is explicit, deterministic, and never described as Source order.
		for _, key := range slices.Sorted(maps.Keys(value)) {
			values = append(values, key)
		}
	default:
		return fail(qosPolicyCloudInvalid("Proxy result %T is not iterable by cloud list()", proxy.Value))
	}
	if err := state.Check(ctx); err != nil {
		return fail(err)
	}
	var encoded json.RawMessage
	if !qosPolicyCloudContainsOpaque(values) {
		encoded, err = json.Marshal(values)
		if err = errors.Join(err, state.Check(ctx)); err != nil {
			return fail(err)
		}
	}
	if err := state.Check(ctx); err != nil {
		return fail(err)
	}
	result.Policies, result.Value, result.JSON = records, values, bytes.Clone(encoded)
	return result, nil
}
