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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstreampolicies "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/fwaas_v2/policies"
)

// FirewallPolicyUpdateOpts owns the pinned six-key cloud update_firewall_policy keyword set. Omitted fields
// remain untouched; nil JSON values explicitly select null.
type FirewallPolicyUpdateOpts struct {
	Attributes map[string]json.RawMessage
	Filters    json.RawMessage
}
type FirewallPolicyUpdateOption func(*FirewallPolicyUpdateOpts) error

func copyFirewallPolicyUpdate(v FirewallPolicyUpdateOpts) FirewallPolicyUpdateOpts {
	out := FirewallPolicyUpdateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes)), Filters: bytes.Clone(v.Filters)}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func firewallPolicyUpdateKey(key string) bool {
	switch key {
	case "audited", "description", "firewall_rules", "name", "project_id", "shared":
		return true
	}
	return false
}
func WithFirewallPolicyUpdateFilters(raw json.RawMessage) FirewallPolicyUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *FirewallPolicyUpdateOpts) error { c.Filters = bytes.Clone(owned); return nil }
}
func WithFirewallPolicyUpdateOpts(v FirewallPolicyUpdateOpts) FirewallPolicyUpdateOption {
	owned := copyFirewallPolicyUpdate(v)
	return func(c *FirewallPolicyUpdateOpts) error { *c = copyFirewallPolicyUpdate(owned); return nil }
}
func WithFirewallPolicyUpdateAttribute(key string, value any) FirewallPolicyUpdateOption {
	if !firewallPolicyUpdateKey(key) {
		return func(*FirewallPolicyUpdateOpts) error {
			return qosReadInvalid("%q is not a pinned cloud firewall policy update keyword", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *FirewallPolicyUpdateOpts) error {
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
func WithFirewallPolicyUpdateJSON(raw json.RawMessage) FirewallPolicyUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *FirewallPolicyUpdateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("firewall policy kwargs require a complete UTF-8 JSON object: %v", err)
		}
		c.Attributes = map[string]json.RawMessage{}
		for _, member := range members {
			c.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithFirewallPolicyUpdateName(v string) FirewallPolicyUpdateOption {
	return WithFirewallPolicyUpdateAttribute("name", v)
}
func WithFirewallPolicyUpdateAudited(v bool) FirewallPolicyUpdateOption {
	return WithFirewallPolicyUpdateAttribute("audited", v)
}
func WithFirewallPolicyUpdateShared(v bool) FirewallPolicyUpdateOption {
	return WithFirewallPolicyUpdateAttribute("shared", v)
}
func WithFirewallPolicyUpdateDescription(v string) FirewallPolicyUpdateOption {
	return WithFirewallPolicyUpdateAttribute("description", v)
}
func WithFirewallPolicyUpdateRules(v ...string) FirewallPolicyUpdateOption {
	return WithFirewallPolicyUpdateAttribute("firewall_rules", append([]string{}, v...))
}

// FirewallPolicyUpdateResult retains the lookup's declared view and wire evidence. Policy is
// the native PUT response only; lookup-only/no-dirty branches never fabricate it.
type FirewallPolicyUpdateResult struct {
	Lookup      *FirewallRecord
	Policy      *policies.Policy
	NoOp        bool
	RuleLookups []*FirewallRecord
}

// firewallPolicyUpdateBody is SDK-owned. Application callers supply concrete
// options, while this private bridge preserves raw kwargs without typed coercion.
type firewallPolicyUpdateBody struct{ attributes map[string]json.RawMessage }

func (v firewallPolicyUpdateBody) ToFirewallPolicyUpdateMap() (map[string]any, error) {
	body := make(map[string]json.RawMessage, len(v.attributes))
	for key, raw := range v.attributes {
		body[key] = bytes.Clone(raw)
	}
	return map[string]any{"firewall_policy": body}, nil
}

// UpdateFirewallPolicyRecord performs the cloud lookup even for empty kwargs. It compares
// supplied values to the looked-up view and commits only changed attributes.
// SDK-owned options replace Python default kwargs without application builders.
func (s *Service) UpdateFirewallPolicyRecord(ctx context.Context, identity string, options ...FirewallPolicyUpdateOption) (*FirewallPolicyUpdateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateFirewallPolicyRecord", "firewall_policies", cloudread.ContextError(ctx, err))
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
	cfg := copyFirewallPolicyUpdate(FirewallPolicyUpdateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil cloud firewall policy update option"))
		}
		next := copyFirewallPolicyUpdate(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyFirewallPolicyUpdate(next)
	}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		if !firewallPolicyUpdateKey(key) {
			return nil, wrap(qosReadInvalid("%q is not a pinned cloud firewall policy update keyword", key))
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
	engine, err := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_policies"), guard)
	if err != nil {
		return nil, wrap(err)
	}

	result := &FirewallPolicyUpdateResult{}
	found, err := engine.Find(ctx, identity, policy)
	result.Lookup = firewallDeleteProof(found, err)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if result.Lookup == nil {
		return result, wrap(&resource.NotFoundError{Resource: "firewall policy", Reference: identity})
	}
	if result.Lookup.Resource == nil || result.Lookup.Wire == nil {
		return result, wrap(qosReadInvalid("firewall policy lookup did not produce a resource view"))
	}
	if cause := firewallDeleteLocation(result.Lookup, "firewall_policies", nil); cause != nil {
		return result, wrap(cause)
	}
	if raw, present := cfg.Attributes["firewall_rules"]; present {
		if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '[' {
			return result, wrap(qosReadInvalid("firewall_rules requires a JSON array of string references"))
		}
		var references []json.RawMessage
		if cause := json.Unmarshal(raw, &references); cause != nil {
			return result, wrap(cause)
		}
		ruleEngine, cause := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_rules"), guard)
		if cause != nil {
			return result, wrap(cause)
		}
		ids := make([]string, 0, len(references))
		for _, reference := range references {
			if cause := guard(ctx); cause != nil {
				return result, wrap(cause)
			}
			var name string
			if json.Unmarshal(reference, &name) != nil || name == "" {
				return result, wrap(qosReadInvalid("firewall rule references must be nonempty strings"))
			}
			rule, cause := ruleEngine.Find(ctx, name, neutronfind.Options{IgnoreMissing: &strict})
			record := firewallDeleteProof(rule, cause)
			if record != nil {
				result.RuleLookups = append(result.RuleLookups, record)
			}
			if cause = errors.Join(cause, guard(ctx)); cause != nil {
				return result, wrap(cause)
			}
			if record == nil || record.Resource == nil {
				return result, wrap(qosReadInvalid("strict firewall rule Find returned no resource"))
			}
			if cause := firewallDeleteLocation(record, "firewall_rules", nil); cause != nil {
				return result, wrap(cause)
			}
			var id string
			if json.Unmarshal(record.Resource.Body["id"], &id) != nil || id == "" {
				return result, wrap((&rest.Response{Body: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}).Fail(qosReadInvalid("firewall rule response requires a nonempty string ID")))
			}
			ids = append(ids, id)
		}
		translated, _ := json.Marshal(ids)
		cfg.Attributes["firewall_rules"] = translated
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
		return result, wrap((&rest.Response{Body: result.Lookup.Envelope, Header: result.Lookup.Header, StatusCode: result.Lookup.StatusCode}).Fail(qosReadInvalid("firewall policy update requires a nonempty string response ID")))
	}
	if err := guard(ctx); err != nil {
		return result, wrap(err)
	}
	mutation := upstreampolicies.Update(ctx, &source.Client, url.PathEscape(id), firewallPolicyUpdateBody{attributes: attrs})
	result.Policy, err = mutation.Extract()
	if err != nil {
		result.Policy = nil // A rejected PUT is not a successful native result.
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
