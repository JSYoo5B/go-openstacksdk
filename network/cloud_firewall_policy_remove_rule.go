package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstreampolicies "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/fwaas_v2/policies"
)

// FirewallPolicyRemoveRuleOpts owns the cloud policy Find filters. They never
// propagate to the rule lookup or action.
type FirewallPolicyRemoveRuleOpts struct{ Filters json.RawMessage }
type FirewallPolicyRemoveRuleOption func(*FirewallPolicyRemoveRuleOpts) error

func WithFirewallPolicyRemoveRuleFilters(raw json.RawMessage) FirewallPolicyRemoveRuleOption {
	owned := bytes.Clone(raw)
	return func(c *FirewallPolicyRemoveRuleOpts) error { c.Filters = bytes.Clone(owned); return nil }
}
func WithFirewallPolicyRemoveRuleOpts(v FirewallPolicyRemoveRuleOpts) FirewallPolicyRemoveRuleOption {
	return WithFirewallPolicyRemoveRuleFilters(v.Filters)
}

// FirewallPolicyRemoveRuleResult separates lookup evidence from the native
// action result. NoOp means no action, not no lookups.
type FirewallPolicyRemoveRuleResult struct {
	Lookup     *FirewallRecord
	RuleLookup *FirewallRecord
	Policy     *policies.Policy
	NoOp       bool
}

// RemoveRuleFromPolicyRecord follows the cloud helper's target-first lookup,
// missing-rule and absent-association short circuits. Applications supply
// concrete options; the SDK owns all Find and action configuration.
func (s *Service) RemoveRuleFromPolicyRecord(ctx context.Context, identity, ruleIdentity string, options ...FirewallPolicyRemoveRuleOption) (*FirewallPolicyRemoveRuleResult, error) {
	wrap := func(err error) error {
		return request.Wrap("RemoveRuleFromPolicyRecord", "firewall_policies", cloudread.ContextError(ctx, err))
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
	cfg := FirewallPolicyRemoveRuleOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil cloud firewall policy remove rule option"))
		}
		next := FirewallPolicyRemoveRuleOpts{Filters: bytes.Clone(cfg.Filters)}
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = FirewallPolicyRemoveRuleOpts{Filters: bytes.Clone(next.Filters)}
	}
	members, err := qosPolicyCloudPairs(cfg.Filters, false)
	if err != nil {
		return nil, wrap(err)
	}
	strict := false
	findOptions := neutronfind.Options{IgnoreMissing: &strict, Query: map[string]json.RawMessage{}}
	for _, member := range members {
		if member.Key == "name_or_id" || member.Key == "ignore_missing" {
			return nil, wrap(qosReadInvalid("%q duplicates cloud Find argument", member.Key))
		}
		findOptions.Query[member.Key] = bytes.Clone(member.Value)
		findOptions.Order = append(findOptions.Order, member.Key)
	}
	engine, err := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_policies"), guard)
	if err != nil {
		return nil, wrap(err)
	}
	result := &FirewallPolicyRemoveRuleResult{}
	found, err := engine.Find(ctx, identity, findOptions)
	result.Lookup = firewallDeleteProof(found, err)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if result.Lookup == nil {
		return result, wrap(&resource.NotFoundError{Resource: "firewall policy", Reference: identity})
	}
	if result.Lookup.Resource == nil {
		return result, wrap(qosReadInvalid("firewall policy lookup has no resource"))
	}
	ruleEngine, err := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_rules"), guard)
	if err != nil {
		return result, wrap(err)
	}
	rule, err := ruleEngine.Find(ctx, ruleIdentity, neutronfind.Options{})
	result.RuleLookup = firewallDeleteProof(rule, err)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if rule == nil {
		result.NoOp = true
		return result, wrap(guard(ctx))
	}
	if result.RuleLookup == nil || result.RuleLookup.Resource == nil {
		return result, wrap(qosReadInvalid("firewall rule lookup has no resource"))
	}
	var ruleID string
	if json.Unmarshal(result.RuleLookup.Resource.Body["id"], &ruleID) != nil || ruleID == "" {
		return result, wrap((&rest.Response{Body: result.RuleLookup.Envelope, Header: result.RuleLookup.Header, StatusCode: result.RuleLookup.StatusCode}).Fail(qosReadInvalid("firewall rule response requires nonempty string ID")))
	}
	var rules []json.RawMessage
	raw := result.Lookup.Resource.Body["firewall_rules"]
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &rules) != nil {
		return result, wrap((&rest.Response{Body: result.Lookup.Envelope, Header: result.Lookup.Header, StatusCode: result.Lookup.StatusCode}).Fail(qosReadInvalid("firewall_rules is not iterable")))
	}
	encoded, _ := json.Marshal(ruleID)
	associated := false
	for _, candidate := range rules {
		same, err := jsonfilter.EqualPythonLoadedJSON(candidate, encoded)
		if err != nil {
			return result, wrap(err)
		}
		if same {
			associated = true
			break
		}
	}
	if !associated {
		result.NoOp = true
		return result, wrap(guard(ctx))
	}
	var policyID string
	if json.Unmarshal(result.Lookup.Resource.Body["id"], &policyID) != nil || policyID == "" {
		return result, wrap(qosReadInvalid("firewall policy action requires nonempty string ID"))
	}
	if err := guard(ctx); err != nil {
		return result, wrap(err)
	}
	mutation := upstreampolicies.RemoveRule(ctx, &source.Client, url.PathEscape(policyID), ruleID)
	result.Policy, err = mutation.Extract()
	if err != nil {
		result.Policy = nil
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
