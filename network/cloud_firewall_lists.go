package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FirewallListOpts owns the cloud filters dictionary of list_firewall_groups,
// list_firewall_policies and list_firewall_rules. Nil and every JSON falsey
// value (null, false, 0, "", [] and {}) mean no filters; a truthy value must be
// a JSON object.
//
// Keys declared in the pinned resource query mapping are sent to Neutron.
// Keys that are only response Body attributes are applied locally to each
// returned row: admin_state_up and id for firewall groups, audited and id for
// policies, id for rules. Any other key is discarded without an error, as
// pinned Resource.list does with unknown parameters.
type FirewallListOpts struct{ Filters json.RawMessage }

// FirewallListOption customizes one firewall list call.
type FirewallListOption func(*FirewallListOpts) error

// WithFirewallListOpts replaces the complete option set with an owned copy.
func WithFirewallListOpts(value FirewallListOpts) FirewallListOption {
	owned := FirewallListOpts{Filters: bytes.Clone(value.Filters)}
	return func(c *FirewallListOpts) error {
		*c = FirewallListOpts{Filters: bytes.Clone(owned.Filters)}
		return nil
	}
}

// WithFirewallListFilters snapshots the filters JSON object.
func WithFirewallListFilters(raw json.RawMessage) FirewallListOption {
	return WithFirewallListOpts(FirewallListOpts{Filters: raw})
}

type firewallListSpec struct {
	family string
	// query keys are sent to the server, limit and marker included.
	query []string
	// local keys are response Body attributes absent from the query mapping.
	local []string
}

var (
	firewallGroupListSpec = firewallListSpec{
		family: "firewall groups",
		query:  []string{"limit", "marker", "description", "egress_firewall_policy_id", "fields", "ingress_firewall_policy_id", "name", "shared", "status", "ports", "project_id"},
		local:  []string{"admin_state_up", "id"},
	}
	firewallPolicyListSpec = firewallListSpec{
		family: "firewall policies",
		query:  []string{"limit", "marker", "description", "fields", "firewall_rules", "name", "project_id", "shared"},
		local:  []string{"audited", "id"},
	}
	firewallRuleListSpec = firewallListSpec{
		family: "firewall rules",
		query: []string{"limit", "marker", "action", "description", "destination_ip_address", "name", "destination_port", "enabled", "fields", "ip_version",
			"project_id", "protocol", "shared", "source_ip_address", "source_port", "firewall_policy_id"},
		local: []string{"id"},
	}
	// Proxy/Resource list controls consumed before filters would reach the server.
	firewallListControls = []string{"paginated", "base_path", "allow_unknown_params", "microversion", "headers", "max_items", "jmespath_filters"}
)

type firewallListPlan struct {
	query url.Values
	local map[string]json.RawMessage
	order []string
}

func firewallListPrepare(spec firewallListSpec, options []FirewallListOption) (*firewallListPlan, error) {
	var cfg FirewallListOpts
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return nil, qosReadInvalid("nil %s list option", spec.family)
		}
		next := FirewallListOpts{Filters: bytes.Clone(cfg.Filters)}
		if err := apply(&next); err != nil {
			return nil, err
		}
		cfg = FirewallListOpts{Filters: bytes.Clone(next.Filters)}
	}
	plan := &firewallListPlan{query: url.Values{}, local: map[string]json.RawMessage{}}
	if cfg.Filters == nil {
		return plan, nil
	}
	truthy, err := cloudfilter.PythonJSONTruthy(cfg.Filters)
	if err != nil {
		return nil, qosReadInvalid("%s filters must contain complete UTF-8 JSON: %v", spec.family, err)
	}
	if !truthy {
		return plan, nil
	}
	members, err := cloudfilter.ObjectMembers(cfg.Filters)
	if err != nil {
		return nil, qosReadInvalid("truthy %s filters must be a JSON object for **filters", spec.family)
	}
	for _, member := range members {
		switch {
		case slices.Contains(firewallListControls, member.Key):
			return nil, qosReadInvalid("%s filter %q is a Proxy/Resource list control and is not supported", spec.family, member.Key)
		case slices.Contains(spec.query, member.Key):
			values, err := cloudfilter.RequestQueryValues(member.Value)
			if err != nil {
				return nil, qosReadInvalid("%s filter %q: %v", spec.family, member.Key, err)
			}
			if len(values) == 0 {
				plan.query.Del(member.Key)
				continue
			}
			plan.query[member.Key] = values
		case slices.Contains(spec.local, member.Key):
			if err := firewallLocalFilterShape(spec, member.Key, member.Value); err != nil {
				return nil, err
			}
			plan.local[member.Key] = bytes.Clone(member.Value)
		}
	}
	plan.order = slices.Sorted(func(yield func(string) bool) {
		for key := range plan.local {
			if !yield(key) {
				return
			}
		}
	})
	return plan, nil
}

func firewallLocalFilterShape(spec firewallListSpec, key string, raw json.RawMessage) error {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return qosReadInvalid("%s filter %q JSON: %v", spec.family, key, err)
	}
	return nil
}

// firewallLocalMatch reproduces the client-side comparison of pinned
// Resource.list for a scalar Body attribute: ordinary Python equality for
// non-object filters, and the nested dictionary helper for object filters.
func firewallLocalMatch(spec firewallListSpec, key string, raw json.RawMessage, actual any) (bool, error) {
	var filter any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&filter); err != nil {
		return false, qosReadInvalid("%s filter %q JSON: %v", spec.family, key, err)
	}
	switch want := filter.(type) {
	case map[string]any:
		truthy := false
		switch value := actual.(type) {
		case bool:
			truthy = value
		case string:
			truthy = value != ""
		}
		if !truthy {
			return false, nil
		}
		if len(want) == 0 {
			return true, nil
		}
		return false, qosReadInvalid("%s filter %q: a nonempty object cannot be compared with a scalar attribute", spec.family, key)
	case bool:
		got, ok := actual.(bool)
		return ok && got == want, nil
	case json.Number:
		got, ok := actual.(bool)
		if !ok {
			return false, nil
		}
		number, err := strconv.ParseFloat(want.String(), 64)
		if err != nil {
			return false, nil
		}
		if got {
			return number == 1, nil
		}
		return number == 0, nil
	case string:
		got, ok := actual.(string)
		return ok && got == want, nil
	default:
		return false, nil
	}
}

func firewallListAll[T any](spec firewallListSpec, plan *firewallListPlan, list func(url.Values) func(func(*T, error) bool), field func(*T, string) any) ([]*T, error) {
	result := make([]*T, 0)
	for row, err := range list(plan.query) {
		if err != nil {
			return nil, err
		}
		matched := true
		for _, key := range plan.order {
			ok, err := firewallLocalMatch(spec, key, plan.local[key], field(row, key))
			if err != nil {
				return nil, err
			}
			if !ok {
				matched = false
				break
			}
		}
		if matched {
			result = append(result, row)
		}
	}
	return result, nil
}

func firewallQueryOption[O any](query url.Values) request.Option[O] {
	return func(config *request.Config[O]) error {
		config.Query = make(url.Values, len(query))
		for key, values := range query {
			config.Query[key] = slices.Clone(values)
		}
		return nil
	}
}

func (s *Service) firewallListSource(op string) error {
	if s == nil || s.API == nil {
		return fmt.Errorf("%w: configured network service is required for %s", resource.ErrInvalidOption, op)
	}
	return nil
}

// ListFirewallGroups implements NetworkCloudMixin.list_firewall_groups: one
// GET fwaas/firewall_groups (following native next links) with the declared
// query keys, then local admin_state_up/id filtering. No extension check is
// made, matching the pinned helper. Rows use the native model, which does not
// retain unknown body attributes or wire presence.
func (s *Service) ListFirewallGroups(ctx context.Context, options ...FirewallListOption) ([]*groups.Group, error) {
	if err := s.firewallListSource("ListFirewallGroups"); err != nil {
		return nil, request.Wrap("ListFirewallGroups", "firewall groups", err)
	}
	plan, err := firewallListPrepare(firewallGroupListSpec, options)
	if err != nil {
		return nil, request.Wrap("ListFirewallGroups", "firewall groups", err)
	}
	api := s.API.FirewallGroups
	rows, err := firewallListAll(firewallGroupListSpec, plan, func(q url.Values) func(func(*groups.Group, error) bool) {
		return api.List(ctx, firewallQueryOption[groups.ListOpts](q))
	}, func(row *groups.Group, key string) any {
		if key == "admin_state_up" {
			return row.AdminStateUp
		}
		return row.ID
	})
	return rows, request.Wrap("ListFirewallGroups", "firewall groups", err)
}

// ListFirewallPolicies implements NetworkCloudMixin.list_firewall_policies
// with local audited/id filtering; see ListFirewallGroups for the shared rules.
func (s *Service) ListFirewallPolicies(ctx context.Context, options ...FirewallListOption) ([]*policies.Policy, error) {
	if err := s.firewallListSource("ListFirewallPolicies"); err != nil {
		return nil, request.Wrap("ListFirewallPolicies", "firewall policies", err)
	}
	plan, err := firewallListPrepare(firewallPolicyListSpec, options)
	if err != nil {
		return nil, request.Wrap("ListFirewallPolicies", "firewall policies", err)
	}
	api := s.API.FirewallPolicies
	rows, err := firewallListAll(firewallPolicyListSpec, plan, func(q url.Values) func(func(*policies.Policy, error) bool) {
		return api.List(ctx, firewallQueryOption[policies.ListOpts](q))
	}, func(row *policies.Policy, key string) any {
		if key == "audited" {
			return row.Audited
		}
		return row.ID
	})
	return rows, request.Wrap("ListFirewallPolicies", "firewall policies", err)
}

// ListFirewallRules implements NetworkCloudMixin.list_firewall_rules with
// local id filtering; see ListFirewallGroups for the shared rules.
func (s *Service) ListFirewallRules(ctx context.Context, options ...FirewallListOption) ([]*rules.Rule, error) {
	if err := s.firewallListSource("ListFirewallRules"); err != nil {
		return nil, request.Wrap("ListFirewallRules", "firewall rules", err)
	}
	plan, err := firewallListPrepare(firewallRuleListSpec, options)
	if err != nil {
		return nil, request.Wrap("ListFirewallRules", "firewall rules", err)
	}
	api := s.API.FirewallRules
	rows, err := firewallListAll(firewallRuleListSpec, plan, func(q url.Values) func(func(*rules.Rule, error) bool) {
		return api.List(ctx, firewallQueryOption[rules.ListOpts](q))
	}, func(row *rules.Rule, _ string) any { return row.ID })
	return rows, request.Wrap("ListFirewallRules", "firewall rules", err)
}
