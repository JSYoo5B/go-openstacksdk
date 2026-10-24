package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FirewallGroupUpdateOpts owns the pinned eight-key cloud update_firewall_group keyword set. Omitted fields
// remain untouched; nil JSON values explicitly select null.
type FirewallGroupUpdateOpts struct {
	Attributes map[string]json.RawMessage
	Filters    json.RawMessage
}
type FirewallGroupUpdateOption func(*FirewallGroupUpdateOpts) error

func copyFirewallGroupUpdate(v FirewallGroupUpdateOpts) FirewallGroupUpdateOpts {
	out := FirewallGroupUpdateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes)), Filters: bytes.Clone(v.Filters)}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func firewallGroupUpdateKey(key string) bool {
	switch key {
	case "admin_state_up", "description", "egress_firewall_policy", "ingress_firewall_policy", "name", "ports", "project_id", "shared":
		return true
	}
	return false
}
func WithFirewallGroupUpdateFilters(raw json.RawMessage) FirewallGroupUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *FirewallGroupUpdateOpts) error { c.Filters = bytes.Clone(owned); return nil }
}
func WithFirewallGroupUpdateOpts(v FirewallGroupUpdateOpts) FirewallGroupUpdateOption {
	owned := copyFirewallGroupUpdate(v)
	return func(c *FirewallGroupUpdateOpts) error { *c = copyFirewallGroupUpdate(owned); return nil }
}
func WithFirewallGroupUpdateAttribute(key string, value any) FirewallGroupUpdateOption {
	if !firewallGroupUpdateKey(key) {
		return func(*FirewallGroupUpdateOpts) error {
			return qosReadInvalid("%q is not a pinned cloud firewall group update keyword", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *FirewallGroupUpdateOpts) error {
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
func WithFirewallGroupUpdateJSON(raw json.RawMessage) FirewallGroupUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *FirewallGroupUpdateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("firewall group kwargs require a complete UTF-8 JSON object: %v", err)
		}
		c.Attributes = map[string]json.RawMessage{}
		for _, member := range members {
			c.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithFirewallGroupUpdateName(v string) FirewallGroupUpdateOption {
	return WithFirewallGroupUpdateAttribute("name", v)
}
func WithFirewallGroupUpdateAdminStateUp(v bool) FirewallGroupUpdateOption {
	return WithFirewallGroupUpdateAttribute("admin_state_up", v)
}
func WithFirewallGroupUpdateShared(v bool) FirewallGroupUpdateOption {
	return WithFirewallGroupUpdateAttribute("shared", v)
}
func WithFirewallGroupUpdateEgressPolicy(v string) FirewallGroupUpdateOption {
	return WithFirewallGroupUpdateAttribute("egress_firewall_policy", v)
}
func WithFirewallGroupUpdateIngressPolicy(v string) FirewallGroupUpdateOption {
	return WithFirewallGroupUpdateAttribute("ingress_firewall_policy", v)
}
func WithFirewallGroupUpdatePorts(v ...string) FirewallGroupUpdateOption {
	return WithFirewallGroupUpdateAttribute("ports", append([]string{}, v...))
}

// FirewallGroupUpdateResult retains the lookup's declared view and wire evidence.
// Response is the owned receipt of the accepted PUT, and Group is the native
// projection of Response.Merged (the Python post-commit resource state).
// Lookup-only/no-dirty branches never fabricate either.
type FirewallGroupUpdateResult struct {
	Lookup        *FirewallRecord
	Group         *groups.Group
	Response      *UpdateReceipt
	NoOp          bool
	PolicyLookups []*FirewallRecord
	PortLookups   []*ports.PortRecord
}

// UpdateFirewallGroupRecord performs the cloud lookup even for empty kwargs. It compares
// supplied values to the looked-up view and commits only changed attributes.
// SDK-owned options replace Python default kwargs without application builders.
func (s *Service) UpdateFirewallGroupRecord(ctx context.Context, identity string, options ...FirewallGroupUpdateOption) (*FirewallGroupUpdateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateFirewallGroupRecord", "firewall_groups", cloudread.ContextError(ctx, err))
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
	cfg := copyFirewallGroupUpdate(FirewallGroupUpdateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil cloud firewall group update option"))
		}
		next := copyFirewallGroupUpdate(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyFirewallGroupUpdate(next)
	}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		if !firewallGroupUpdateKey(key) {
			return nil, wrap(qosReadInvalid("%q is not a pinned cloud firewall group update keyword", key))
		}
		if cfg.Attributes[key] == nil {
			cfg.Attributes[key] = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(cfg.Attributes[key]); err != nil {
			return nil, wrap(qosReadInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err))
		}
	}
	strict := false
	policy, err := firewallFindOptions(cfg.Filters, nil, nil)
	if err != nil {
		return nil, wrap(err)
	}
	engine, err := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_groups"), guard)
	if err != nil {
		return nil, wrap(err)
	}

	result := &FirewallGroupUpdateResult{}
	found, err := engine.FindWithFetchControls(ctx, identity, policy)
	result.Lookup = firewallDeleteProof(found, err)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if result.Lookup == nil {
		return result, wrap(&resource.NotFoundError{Resource: "firewall group", Reference: identity})
	}
	if result.Lookup.Resource == nil || result.Lookup.Wire == nil {
		return result, wrap(qosReadInvalid("firewall group lookup did not produce a resource view"))
	}
	if cause := firewallDeleteLocation(result.Lookup, "firewall_groups", nil); cause != nil {
		return result, wrap(cause)
	}
	policyEngine, cause := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_policies"), guard)
	if cause != nil {
		return result, wrap(cause)
	}
	for _, key := range []string{"egress_firewall_policy", "ingress_firewall_policy"} {
		raw, present := cfg.Attributes[key]
		if !present {
			continue
		}
		value := json.RawMessage("null")
		if !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			var name string
			if json.Unmarshal(raw, &name) != nil || name == "" {
				return result, wrap(qosReadInvalid("firewall policy reference requires a nonempty string or null"))
			}
			policy, cause := policyEngine.Find(ctx, name, neutronfind.Options{IgnoreMissing: &strict})
			record := firewallDeleteProof(policy, cause)
			if record != nil {
				result.PolicyLookups = append(result.PolicyLookups, record)
			}
			if cause = errors.Join(cause, guard(ctx)); cause != nil {
				return result, wrap(cause)
			}
			if record == nil || record.Resource == nil {
				return result, wrap(qosReadInvalid("strict firewall policy Find returned no resource"))
			}
			if cause := firewallDeleteLocation(record, "firewall_policies", nil); cause != nil {
				return result, wrap(cause)
			}
			var id string
			if json.Unmarshal(record.Resource.Body["id"], &id) != nil || id == "" {
				return result, wrap((&rest.Response{Body: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}).Fail(qosReadInvalid("firewall policy response requires a nonempty string ID")))
			}
			value = bytes.Clone(record.Resource.Body["id"])
		}
		cfg.Attributes[key+"_id"] = value
		delete(cfg.Attributes, key)
	}
	if raw, present := cfg.Attributes["ports"]; present {
		if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '[' {
			return result, wrap(qosReadInvalid("ports requires a JSON array of string references"))
		}
		var references []json.RawMessage
		if cause := json.Unmarshal(raw, &references); cause != nil {
			return result, wrap(cause)
		}
		ids := make([]string, 0, len(references))
		portAPI := ports.New(&source.Client)
		for _, reference := range references {
			if cause := guard(ctx); cause != nil {
				return result, wrap(cause)
			}
			var name string
			if json.Unmarshal(reference, &name) != nil || name == "" {
				return result, wrap(qosReadInvalid("port references must be nonempty strings"))
			}
			port, cause := portAPI.FindRecord(ctx, name)
			if port != nil {
				result.PortLookups = append(result.PortLookups, port)
			}
			if cause = errors.Join(cause, guard(ctx)); cause != nil {
				return result, wrap(cause)
			}
			if port == nil {
				return result, wrap(&resource.NotFoundError{Resource: "port", Reference: name})
			}
			var id string
			if port.Resource == nil || json.Unmarshal(port.Resource.Body["id"], &id) != nil || id == "" {
				return result, wrap((&rest.Response{Body: port.Envelope, Header: port.Header, StatusCode: port.StatusCode}).Fail(qosReadInvalid("port response requires a nonempty string ID")))
			}
			ids = append(ids, id)
		}
		translated, _ := json.Marshal(ids)
		cfg.Attributes["ports"] = translated
	}
	lookup := cloudUpdateSource(found.Source().Body(), found.Source().Translate, result.Lookup.Wire)
	attrs := map[string]json.RawMessage{}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		raw := cfg.Attributes[key]
		old, exists := lookup.state[key]
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
		return result, wrap((&rest.Response{Body: result.Lookup.Envelope, Header: result.Lookup.Header, StatusCode: result.Lookup.StatusCode}).Fail(qosReadInvalid("firewall group update requires a nonempty string response ID")))
	}
	if err := guard(ctx); err != nil {
		return result, wrap(err)
	}
	result.Response, err = cloudUpdatePut(ctx, &source.Client, guard, source.Client.ServiceURL("fwaas", "firewall_groups", url.PathEscape(id)), "firewall_group", lookup, cfg.Attributes, attrs)
	if err == nil {
		result.Group, err = decodeUpdateMerged[groups.Group](result.Response)
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
