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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FirewallRuleUpdateOpts owns the pinned fourteen-key cloud update_firewall_rule keyword set. Omitted fields
// remain untouched; nil JSON values explicitly select null.
type FirewallRuleUpdateOpts struct {
	Attributes map[string]json.RawMessage
	Filters    json.RawMessage
}
type FirewallRuleUpdateOption func(*FirewallRuleUpdateOpts) error

func copyFirewallRuleUpdate(v FirewallRuleUpdateOpts) FirewallRuleUpdateOpts {
	out := FirewallRuleUpdateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes)), Filters: bytes.Clone(v.Filters)}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func firewallRuleUpdateKey(key string) bool {
	switch key {
	case "action", "description", "destination_firewall_group_id", "destination_ip_address", "destination_port", "enabled", "ip_version", "name", "project_id", "protocol", "shared", "source_firewall_group_id", "source_ip_address", "source_port":
		return true
	}
	return false
}
func WithFirewallRuleUpdateFilters(raw json.RawMessage) FirewallRuleUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *FirewallRuleUpdateOpts) error { c.Filters = bytes.Clone(owned); return nil }
}
func WithFirewallRuleUpdateOpts(v FirewallRuleUpdateOpts) FirewallRuleUpdateOption {
	owned := copyFirewallRuleUpdate(v)
	return func(c *FirewallRuleUpdateOpts) error { *c = copyFirewallRuleUpdate(owned); return nil }
}
func WithFirewallRuleUpdateAttribute(key string, value any) FirewallRuleUpdateOption {
	if !firewallRuleUpdateKey(key) {
		return func(*FirewallRuleUpdateOpts) error {
			return qosReadInvalid("%q is not a pinned cloud firewall rule update keyword", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *FirewallRuleUpdateOpts) error {
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
func WithFirewallRuleUpdateJSON(raw json.RawMessage) FirewallRuleUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *FirewallRuleUpdateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("firewall rule kwargs require a complete UTF-8 JSON object: %v", err)
		}
		c.Attributes = map[string]json.RawMessage{}
		for _, member := range members {
			c.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithFirewallRuleUpdateName(v string) FirewallRuleUpdateOption {
	return WithFirewallRuleUpdateAttribute("name", v)
}
func WithFirewallRuleUpdateAction(v string) FirewallRuleUpdateOption {
	return WithFirewallRuleUpdateAttribute("action", v)
}
func WithFirewallRuleUpdateEnabled(v bool) FirewallRuleUpdateOption {
	return WithFirewallRuleUpdateAttribute("enabled", v)
}
func WithFirewallRuleUpdateShared(v bool) FirewallRuleUpdateOption {
	return WithFirewallRuleUpdateAttribute("shared", v)
}
func WithFirewallRuleUpdateProtocol(v string) FirewallRuleUpdateOption {
	return WithFirewallRuleUpdateAttribute("protocol", v)
}

// FirewallRuleUpdateResult retains the lookup's declared view and wire evidence.
// Response is the owned receipt of the accepted PUT, and Rule is the native
// projection of Response.Merged (the Python post-commit resource state).
// Lookup-only/no-dirty branches never fabricate either.
type FirewallRuleUpdateResult struct {
	Lookup   *FirewallRecord
	Rule     *rules.Rule
	Response *UpdateReceipt
	NoOp     bool
}

// UpdateFirewallRuleRecord performs the cloud lookup even for empty kwargs. It compares
// supplied values to the looked-up view and commits only changed attributes.
// SDK-owned options replace Python default kwargs without application builders.
func (s *Service) UpdateFirewallRuleRecord(ctx context.Context, identity string, options ...FirewallRuleUpdateOption) (*FirewallRuleUpdateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateFirewallRuleRecord", "firewall_rules", cloudread.ContextError(ctx, err))
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
	cfg := copyFirewallRuleUpdate(FirewallRuleUpdateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil cloud firewall rule update option"))
		}
		next := copyFirewallRuleUpdate(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyFirewallRuleUpdate(next)
	}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		if !firewallRuleUpdateKey(key) {
			return nil, wrap(qosReadInvalid("%q is not a pinned cloud firewall rule update keyword", key))
		}
		if cfg.Attributes[key] == nil {
			cfg.Attributes[key] = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(cfg.Attributes[key]); err != nil {
			return nil, wrap(qosReadInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err))
		}
	}
	members, err := qosPolicyCloudPairs(cfg.Filters, false)
	if err != nil {
		return nil, wrap(err)
	}
	strict := false
	policy := neutronfind.Options{IgnoreMissing: &strict, Query: map[string]json.RawMessage{}}
	for _, member := range members {
		if member.Key == "name_or_id" || member.Key == "ignore_missing" {
			return nil, wrap(qosReadInvalid("%q duplicates cloud Find argument", member.Key))
		}
		policy.Query[member.Key] = bytes.Clone(member.Value)
		policy.Order = append(policy.Order, member.Key)
	}
	engine, err := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_rules"), guard)
	if err != nil {
		return nil, wrap(err)
	}

	result := &FirewallRuleUpdateResult{}
	found, err := engine.Find(ctx, identity, policy)
	result.Lookup = firewallDeleteProof(found, err)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if result.Lookup == nil {
		return result, wrap(&resource.NotFoundError{Resource: "firewall rule", Reference: identity})
	}
	if result.Lookup.Resource == nil || result.Lookup.Wire == nil {
		return result, wrap(qosReadInvalid("firewall rule lookup did not produce a resource view"))
	}
	if cause := firewallDeleteLocation(result.Lookup, "firewall_rules", nil); cause != nil {
		return result, wrap(cause)
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
		return result, wrap((&rest.Response{Body: result.Lookup.Envelope, Header: result.Lookup.Header, StatusCode: result.Lookup.StatusCode}).Fail(qosReadInvalid("firewall rule update requires a nonempty string response ID")))
	}
	if err := guard(ctx); err != nil {
		return result, wrap(err)
	}
	result.Response, err = cloudUpdatePut(ctx, &source.Client, guard, source.Client.ServiceURL("fwaas", "firewall_rules", url.PathEscape(id)), "firewall_rule", lookup, cfg.Attributes, attrs)
	if err == nil {
		result.Rule, err = decodeUpdateMerged[rules.Rule](result.Response)
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
