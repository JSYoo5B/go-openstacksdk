package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// QoSPolicyMutationOpts is the Source cloud kwargs domain. Omitted and explicit
// null/default/false/zero remain separate. The SDK owns validation/translation.
type QoSPolicyMutationOpts struct{ Attributes map[string]json.RawMessage }
type QoSPolicyMutationOption func(*QoSPolicyMutationOpts) error

func qosPolicyMutationKey(key string) bool {
	switch key {
	case "name", "description", "shared", "default", "project_id":
		return true
	}
	return false
}
func copyQoSPolicyMutationOpts(value QoSPolicyMutationOpts) QoSPolicyMutationOpts {
	out := QoSPolicyMutationOpts{Attributes: make(map[string]json.RawMessage, len(value.Attributes))}
	for key, raw := range value.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func WithQoSPolicyMutationOpts(value QoSPolicyMutationOpts) QoSPolicyMutationOption {
	owned := copyQoSPolicyMutationOpts(value)
	return func(current *QoSPolicyMutationOpts) error { *current = copyQoSPolicyMutationOpts(owned); return nil }
}

// WithQoSPolicyAttribute rejects unknown cloud kwargs before marshaling values.
// Dedicated convenience functions below need no application builder/interface.
func WithQoSPolicyAttribute(key string, value any) QoSPolicyMutationOption {
	if !qosPolicyMutationKey(key) {
		return func(*QoSPolicyMutationOpts) error {
			return qosPolicyCloudInvalid("%q is not a cloud policy keyword", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(current *QoSPolicyMutationOpts) error {
		if err != nil {
			return qosPolicyCloudInvalid("attribute %q JSON: %v", key, err)
		}
		if current.Attributes == nil {
			current.Attributes = make(map[string]json.RawMessage)
		}
		current.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func WithQoSPolicyMutationJSON(raw json.RawMessage) QoSPolicyMutationOption {
	owned := bytes.Clone(raw)
	return func(current *QoSPolicyMutationOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosPolicyCloudInvalid("mutation kwargs must be a complete UTF-8 JSON object: %v", err)
		}
		current.Attributes = make(map[string]json.RawMessage, len(members))
		for _, member := range members {
			current.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithQoSPolicyName(value string) QoSPolicyMutationOption {
	return WithQoSPolicyAttribute("name", value)
}
func WithQoSPolicyDescription(value string) QoSPolicyMutationOption {
	return WithQoSPolicyAttribute("description", value)
}
func WithQoSPolicyShared(value bool) QoSPolicyMutationOption {
	return WithQoSPolicyAttribute("shared", value)
}
func WithQoSPolicyDefault(value bool) QoSPolicyMutationOption {
	return WithQoSPolicyAttribute("default", value)
}
func WithQoSPolicyProjectID(value string) QoSPolicyMutationOption {
	return WithQoSPolicyAttribute("project_id", value)
}

// QoSPolicyMutation separates lookup and mutation evidence. NoOp means Source's
// empty-after-default or equal-dirty branch; it never creates PUT evidence.
// DefaultDropped records a nonnull default removed when qos-default is absent.
type QoSPolicyMutation struct {
	Policy, Lookup       *policies.PolicyRecord
	Extensions           []*extensions.ExtensionRecord
	NoOp, DefaultDropped bool
}

func (s *Service) qosMutationPrepare(ctx context.Context, options []QoSPolicyMutationOption) (context.Context, *Service, QoSPolicyMutationOpts, error) {
	value := copyQoSPolicyMutationOpts(QoSPolicyMutationOpts{})
	if err := cloudread.Context(ctx); err != nil {
		return ctx, nil, value, err
	}
	if s == nil || s.API == nil || s.client == nil || s.API.RawClient() != s.client {
		return ctx, nil, value, qosPolicyCloudInvalid("configured network service/API is required")
	}
	selectedAPI, selectedClient := s.API, s.client
	source, err := cloudread.Capture(ctx, selectedClient, "network")
	if err != nil {
		return ctx, nil, value, err
	}
	guard := func(ctx context.Context) error {
		if s.API != selectedAPI || s.client != selectedClient || selectedAPI.RawClient() != selectedClient {
			return qosPolicyCloudInvalid("selected network service/API changed")
		}
		return source.Guard(ctx)
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return ctx, nil, value, err
		}
		if apply == nil {
			return ctx, nil, value, qosPolicyCloudInvalid("nil QoS mutation option")
		}
		next := copyQoSPolicyMutationOpts(value)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return ctx, nil, value, err
		}
		value = copyQoSPolicyMutationOpts(next)
	}
	// Source valid_kwargs checks the entire keyword set before any extension GET.
	for _, key := range slices.Sorted(maps.Keys(value.Attributes)) {
		if !qosPolicyMutationKey(key) {
			return ctx, nil, value, qosPolicyCloudInvalid("%q is not a cloud policy keyword", key)
		}
		if value.Attributes[key] == nil {
			value.Attributes[key] = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(value.Attributes[key]); err != nil {
			return ctx, nil, value, qosPolicyCloudInvalid("attribute %q must contain complete UTF-8 JSON: %v", key, err)
		}
	}
	private := &Service{client: &source.Client, API: networkv2.New(&source.Client)}
	return ctx, private, value, nil
}
func qosPolicyMutationGate(ctx context.Context, selected *Service, input QoSPolicyMutationOpts) (*qosReadState, *QoSPolicyMutation, map[string]json.RawMessage, error) {
	result := &QoSPolicyMutation{}
	state, err := qosReadStart(ctx, selected, "qos")
	if state != nil {
		result.Extensions = append(result.Extensions, state.Extensions...)
	}
	if err != nil {
		return state, result, nil, err
	}
	attrs := copyQoSPolicyMutationOpts(input).Attributes
	defaultValue, hasDefault := attrs["default"]
	delete(attrs, "default")
	if hasDefault && !bytes.Equal(bytes.TrimSpace(defaultValue), []byte("null")) {
		optional, err := qosReadStart(ctx, selected, "qos-default")
		if optional != nil {
			result.Extensions = append(result.Extensions, optional.Extensions...)
		}
		if err != nil {
			if !errors.Is(err, ErrUnavailableExtension) {
				return state, result, nil, err
			}
			result.DefaultDropped = true
		} else {
			attrs["is_default"] = bytes.Clone(defaultValue)
		}
	}
	return state, result, attrs, nil
}

// CreateQoSPolicyRecord validates the cloud kwargs before the full qos gate,
// performs a fresh qos-default gate for every nonnull default and always POSTs
// the translated kwargs, including an empty policy body.
func (s *Service) CreateQoSPolicyRecord(ctx context.Context, options ...QoSPolicyMutationOption) (*QoSPolicyMutation, error) {
	const op = "CreateQoSPolicyRecord"
	ctx, selected, input, err := s.qosMutationPrepare(ctx, options)
	if err != nil {
		return nil, qosPolicyCloudWrap(ctx, op, err)
	}
	state, result, attrs, err := qosPolicyMutationGate(ctx, selected, input)
	if err != nil {
		return result, qosPolicyCloudWrap(ctx, op, err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	result.Policy, err = state.API.QoSPolicies.CreateCloudRecord(ctx, attrs)
	return result, qosPolicyCloudWrap(ctx, op, errors.Join(err, state.Check(ctx)))
}

// UpdateQoSPolicyRecord drops unsupported default first. Empty translated attrs
// return None/no Find/no PUT; otherwise default Find must succeed and only raw
// Source-dirty fields are PUT. No general get_qos_policy raw kwargs are added.
func (s *Service) UpdateQoSPolicyRecord(ctx context.Context, identity string, options ...QoSPolicyMutationOption) (*QoSPolicyMutation, error) {
	const op = "UpdateQoSPolicyRecord"
	ctx, selected, input, err := s.qosMutationPrepare(ctx, options)
	if err != nil {
		return nil, qosPolicyCloudWrap(ctx, op, err)
	}
	state, result, attrs, err := qosPolicyMutationGate(ctx, selected, input)
	if err != nil {
		return result, qosPolicyCloudWrap(ctx, op, err)
	}
	if len(attrs) == 0 {
		result.NoOp = true
		return result, qosPolicyCloudWrap(ctx, op, state.Check(ctx))
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	result.Lookup, err = state.API.QoSPolicies.FindCloudMutationRecord(ctx, identity)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return result, qosPolicyCloudWrap(ctx, op, err)
	}
	if result.Lookup == nil {
		return result, qosPolicyCloudWrap(ctx, op, &resource.NotFoundError{Resource: "QoS policy", Reference: identity})
	}
	var changed bool
	result.Policy, changed, err = state.API.QoSPolicies.UpdateCloudRecord(ctx, result.Lookup, attrs)
	result.NoOp = !changed && err == nil
	return result, qosPolicyCloudWrap(ctx, op, errors.Join(err, state.Check(ctx)))
}
