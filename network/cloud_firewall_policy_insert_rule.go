package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FirewallPolicyInsertOpts owns filters and optional position references.
// Omitted/null positions skip their lookup. Position parsing follows membership.
type FirewallPolicyInsertOpts struct{ Filters, After, Before json.RawMessage }
type FirewallPolicyInsertOption func(*FirewallPolicyInsertOpts) error

func copyFirewallPolicyInsert(v FirewallPolicyInsertOpts) FirewallPolicyInsertOpts {
	return FirewallPolicyInsertOpts{Filters: bytes.Clone(v.Filters), After: bytes.Clone(v.After), Before: bytes.Clone(v.Before)}
}
func WithFirewallPolicyInsertOpts(v FirewallPolicyInsertOpts) FirewallPolicyInsertOption {
	owned := copyFirewallPolicyInsert(v)
	return func(c *FirewallPolicyInsertOpts) error { *c = copyFirewallPolicyInsert(owned); return nil }
}
func WithFirewallPolicyInsertFilters(v json.RawMessage) FirewallPolicyInsertOption {
	owned := bytes.Clone(v)
	return func(c *FirewallPolicyInsertOpts) error { c.Filters = bytes.Clone(owned); return nil }
}
func WithFirewallPolicyInsertAfterJSON(v json.RawMessage) FirewallPolicyInsertOption {
	owned := bytes.Clone(v)
	return func(c *FirewallPolicyInsertOpts) error { c.After = bytes.Clone(owned); return nil }
}
func WithFirewallPolicyInsertBeforeJSON(v json.RawMessage) FirewallPolicyInsertOption {
	owned := bytes.Clone(v)
	return func(c *FirewallPolicyInsertOpts) error { c.Before = bytes.Clone(owned); return nil }
}
func WithFirewallPolicyInsertAfter(v string) FirewallPolicyInsertOption {
	raw, _ := json.Marshal(v)
	return WithFirewallPolicyInsertAfterJSON(raw)
}
func WithFirewallPolicyInsertBefore(v string) FirewallPolicyInsertOption {
	raw, _ := json.Marshal(v)
	return WithFirewallPolicyInsertBeforeJSON(raw)
}

// FirewallPolicyInsertion separates lookup receipts from native action output.
type FirewallPolicyInsertion struct {
	Lookup, Rule *FirewallRecord
	Positions    []*FirewallRecord
	Action       *policies.PolicyRecord
	NoOp         bool
}

func firewallInsertID(record *FirewallRecord) (string, error) {
	var id string
	if record == nil || record.Resource == nil || json.Unmarshal(record.Resource.Body["id"], &id) != nil || id == "" {
		cause := qosReadInvalid("firewall insertion requires a nonempty string response ID")
		if record != nil {
			return "", (&rest.Response{Body: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}).Fail(cause)
		}
		return "", cause
	}
	return id, nil
}

// InsertFirewallRuleIntoPolicyRecord resolves policy then rule. Already-present
// rules return the lookup without interpreting positions or sending an action.
func (s *Service) InsertFirewallRuleIntoPolicyRecord(ctx context.Context, identity, ruleIdentity string, options ...FirewallPolicyInsertOption) (*FirewallPolicyInsertion, error) {
	wrap := func(err error) error {
		return request.Wrap("InsertFirewallRuleIntoPolicyRecord", "firewall_policies", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if s == nil || s.API == nil || s.client == nil || s.API.RawClient() != s.client {
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
	cfg := FirewallPolicyInsertOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil firewall insertion option"))
		}
		next := copyFirewallPolicyInsert(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyFirewallPolicyInsert(next)
	}
	pairs, err := qosPolicyCloudPairs(cfg.Filters, false)
	if err != nil {
		return nil, wrap(err)
	}
	strict := false
	findOpts := neutronfind.Options{IgnoreMissing: &strict, Query: map[string]json.RawMessage{}}
	for _, p := range pairs {
		if p.Key == "name_or_id" || p.Key == "ignore_missing" {
			return nil, wrap(qosReadInvalid("%q duplicates cloud Find argument", p.Key))
		}
		findOpts.Query[p.Key] = bytes.Clone(p.Value)
		findOpts.Order = append(findOpts.Order, p.Key)
	}
	policyEngine, err := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_policies"), guard)
	if err != nil {
		return nil, wrap(err)
	}
	result := &FirewallPolicyInsertion{}
	found, err := policyEngine.Find(ctx, identity, findOpts)
	result.Lookup = firewallDeleteProof(found, err)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if result.Lookup == nil || result.Lookup.Resource == nil {
		return result, wrap(&resource.NotFoundError{Resource: "firewall policy", Reference: identity})
	}
	if err := firewallDeleteLocation(result.Lookup, "firewall_policies", nil); err != nil {
		return result, wrap(err)
	}
	ruleEngine, err := neutronfind.New(ctx, &source.Client, firewallDeleteProfile("firewall_rules"), guard)
	if err != nil {
		return result, wrap(err)
	}
	findRule := func(name string) (*FirewallRecord, error) {
		found, cause := ruleEngine.Find(ctx, name, neutronfind.Options{IgnoreMissing: &strict})
		record := firewallDeleteProof(found, cause)
		if cause = errors.Join(cause, guard(ctx)); cause != nil {
			return record, cause
		}
		if record == nil || record.Resource == nil {
			return record, &resource.NotFoundError{Resource: "firewall rule", Reference: name}
		}
		return record, firewallDeleteLocation(record, "firewall_rules", nil)
	}
	result.Rule, err = findRule(ruleIdentity)
	if err != nil {
		return result, wrap(err)
	}
	ruleID, err := firewallInsertID(result.Rule)
	if err != nil {
		return result, wrap(err)
	}
	rawIDs := result.Lookup.Resource.Body["firewall_rules"]
	var ids []json.RawMessage
	if len(bytes.TrimSpace(rawIDs)) == 0 || bytes.TrimSpace(rawIDs)[0] != '[' || json.Unmarshal(rawIDs, &ids) != nil {
		return result, wrap((&rest.Response{Body: result.Lookup.Envelope, Header: result.Lookup.Header, StatusCode: result.Lookup.StatusCode}).Fail(qosReadInvalid("firewall policy membership requires a JSON list")))
	}
	for _, id := range ids {
		same, cause := jsonfilter.EqualPythonLoadedJSON(id, result.Rule.Resource.Body["id"])
		if cause != nil {
			return result, wrap(cause)
		}
		if same {
			result.NoOp = true
			return result, wrap(guard(ctx))
		}
	}
	input := policies.InsertRuleRecordOpts{}
	for i, raw := range []json.RawMessage{cfg.After, cfg.Before} {
		if raw == nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		var name string
		if json.Unmarshal(raw, &name) != nil || name == "" {
			return result, wrap(qosReadInvalid("position reference requires a nonempty JSON string or null"))
		}
		record, cause := findRule(name)
		if record != nil {
			result.Positions = append(result.Positions, record)
		}
		if cause != nil {
			return result, wrap(cause)
		}
		id, cause := firewallInsertID(record)
		if cause != nil {
			return result, wrap(cause)
		}
		if i == 0 {
			input.InsertAfter = &id
		} else {
			input.InsertBefore = &id
		}
	}
	policyID, err := firewallInsertID(result.Lookup)
	if err != nil {
		return result, wrap(err)
	}
	if err := guard(ctx); err != nil {
		return result, wrap(err)
	}
	result.Action, err = policies.New(&source.Client).InsertRuleRecord(ctx, policyID, ruleID, policies.WithInsertRuleRecordOptions(input))
	if err != nil {
		if result.Action != nil {
			result.Action.Policy = nil
		}
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
