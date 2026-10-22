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
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PortUpdateOpts owns the pinned cloud update_port keyword set. Omitted fields
// remain untouched; nil JSON values explicitly select null.
type PortUpdateOpts struct{ Attributes map[string]json.RawMessage }
type PortUpdateOption func(*PortUpdateOpts) error

func copyPortUpdate(v PortUpdateOpts) PortUpdateOpts {
	out := PortUpdateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes))}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func portUpdateKey(key string) bool {
	switch key {
	case "name", "admin_state_up", "fixed_ips", "security_groups", "allowed_address_pairs", "extra_dhcp_opts", "device_owner", "device_id", "binding:vnic_type", "binding:profile", "port_security_enabled", "qos_policy_id", "binding:host_id":
		return true
	}
	return false
}
func WithPortUpdateOpts(v PortUpdateOpts) PortUpdateOption {
	owned := copyPortUpdate(v)
	return func(c *PortUpdateOpts) error { *c = copyPortUpdate(owned); return nil }
}
func WithPortUpdateAttribute(key string, value any) PortUpdateOption {
	if !portUpdateKey(key) {
		return func(*PortUpdateOpts) error {
			return qosReadInvalid("%q is not a pinned cloud port update keyword", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *PortUpdateOpts) error {
		if err != nil {
			return qosReadInvalid("attribute %q JSON: %v", key, err)
		}
		if c.Attributes == nil {
			c.Attributes = map[string]json.RawMessage{}
		}
		c.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func WithPortUpdateJSON(raw json.RawMessage) PortUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *PortUpdateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("port kwargs require a complete UTF-8 JSON object: %v", err)
		}
		c.Attributes = map[string]json.RawMessage{}
		for _, member := range members {
			c.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithPortUpdateName(v string) PortUpdateOption { return WithPortUpdateAttribute("name", v) }
func WithPortUpdateAdminStateUp(v bool) PortUpdateOption {
	return WithPortUpdateAttribute("admin_state_up", v)
}
func WithPortUpdateSecurityGroups(v ...string) PortUpdateOption {
	return WithPortUpdateAttribute("security_groups", append([]string{}, v...))
}
func WithPortUpdatePortSecurityEnabled(v bool) PortUpdateOption {
	return WithPortUpdateAttribute("port_security_enabled", v)
}
func WithPortUpdateDeviceID(v string) PortUpdateOption {
	return WithPortUpdateAttribute("device_id", v)
}
func WithPortUpdateDeviceOwner(v string) PortUpdateOption {
	return WithPortUpdateAttribute("device_owner", v)
}
func WithPortUpdateQoSPolicyID(v string) PortUpdateOption {
	return WithPortUpdateAttribute("qos_policy_id", v)
}

// PortUpdateResult retains the lookup's declared view and wire evidence.
// Response is the owned receipt of the accepted PUT, and Port is the native
// projection of Response.Merged (the Python post-commit resource state);
// Response.Merged also keeps extension attributes absent from Port.
// Lookup-only/no-dirty branches never fabricate either.
type PortUpdateResult struct {
	Lookup   *ports.PortRecord
	Port     *ports.Port
	Response *UpdateReceipt
	NoOp     bool
}

// UpdatePortRecord performs the cloud lookup even for empty kwargs. It compares
// supplied values to the looked-up view and commits only changed attributes.
// SDK-owned options replace Python default kwargs without application builders.
func (s *Service) UpdatePortRecord(ctx context.Context, identity string, options ...PortUpdateOption) (*PortUpdateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdatePortRecord", "ports", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if s == nil || s.client == nil || s.API == nil || s.API.RawClient() != s.client {
		return nil, wrap(qosReadInvalid("configured network service/API required"))
	}
	selectedAPI, selectedClient := s.API, s.client
	source, err := cloudread.Capture(ctx, selectedClient, "network")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error {
		if s.API != selectedAPI || s.client != selectedClient || selectedAPI.RawClient() != selectedClient {
			return qosReadInvalid("selected network service/API changed")
		}
		return source.Guard(ctx)
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	cfg := copyPortUpdate(PortUpdateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil cloud port update option"))
		}
		next := copyPortUpdate(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyPortUpdate(next)
	}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		if !portUpdateKey(key) {
			return nil, wrap(qosReadInvalid("%q is not a pinned cloud port update keyword", key))
		}
		if cfg.Attributes[key] == nil {
			cfg.Attributes[key] = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(cfg.Attributes[key]); err != nil {
			return nil, wrap(qosReadInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err))
		}
	}
	selected := networkv2.New(&source.Client)
	result := &PortUpdateResult{}
	result.Lookup, err = selected.Ports.FindRecord(ctx, identity)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if result.Lookup == nil {
		return result, wrap(&resource.NotFoundError{Resource: "port", Reference: identity})
	}
	if result.Lookup.Resource == nil || result.Lookup.Wire == nil {
		return result, wrap(qosReadInvalid("port lookup did not produce a resource view"))
	}
	attrs := map[string]json.RawMessage{}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		raw := cfg.Attributes[key]
		old, exists := result.Lookup.Wire.Body[key]
		if exists {
			same, e := jsonfilter.EqualPythonLoadedJSON(old, raw)
			if e != nil {
				return result, wrap(e)
			}
			if same {
				continue
			}
		}
		attrs[key] = bytes.Clone(raw)
	}
	if len(attrs) == 0 {
		result.NoOp = true
		return result, wrap(guard(ctx))
	}
	var id string
	if result.Lookup.Resource == nil || json.Unmarshal(result.Lookup.Resource.Body["id"], &id) != nil || id == "" {
		return result, wrap(routerInterfaceProof(result.Lookup, qosReadInvalid("port update requires a nonempty string response ID")))
	}
	if err := guard(ctx); err != nil {
		return result, wrap(err)
	}
	result.Response, err = cloudUpdatePut(ctx, &source.Client, guard, source.Client.ServiceURL("ports", id), "port", result.Lookup.Wire.Body, cfg.Attributes, attrs)
	if err == nil {
		result.Port, err = decodeUpdateMerged[ports.Port](result.Response)
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
