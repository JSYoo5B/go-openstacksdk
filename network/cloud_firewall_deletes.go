package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FirewallDeleteOpts owns cloud filters and optional Go transport/location facts.
// Filters are Source Find kwargs, not a local JMESPath expression. Both Find
// and DELETE are strict; only a clean NotFound is returned as Deleted=false.
type FirewallDeleteOpts struct {
	Filters         json.RawMessage
	Headers         map[string]string
	Microversion    *string
	CurrentLocation *resource.CloudLocation
}
type FirewallDeleteOption func(*FirewallDeleteOpts) error

func copyFirewallDelete(v FirewallDeleteOpts) FirewallDeleteOpts {
	v.Filters = bytes.Clone(v.Filters)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.CurrentLocation != nil {
		x := v.CurrentLocation.Clone()
		v.CurrentLocation = &x
	}
	return v
}
func WithFirewallDeleteOpts(v FirewallDeleteOpts) FirewallDeleteOption {
	owned := copyFirewallDelete(v)
	return func(c *FirewallDeleteOpts) error { *c = copyFirewallDelete(owned); return nil }
}
func WithFirewallDeleteFilters(v json.RawMessage) FirewallDeleteOption {
	owned := bytes.Clone(v)
	return func(c *FirewallDeleteOpts) error { c.Filters = bytes.Clone(owned); return nil }
}
func WithFirewallDeleteHeaders(v map[string]string) FirewallDeleteOption {
	owned := maps.Clone(v)
	return func(c *FirewallDeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithFirewallDeleteMicroversion(v string) FirewallDeleteOption {
	return func(c *FirewallDeleteOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithFirewallDeleteCurrentLocation(v resource.CloudLocation) FirewallDeleteOption {
	owned := v.Clone()
	return func(c *FirewallDeleteOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}

// FirewallRecord contains the Source view and independent physical lookup proof.
// A metadata-only record on a physical failure has no Resource/Wire.
type FirewallRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}
type FirewallDeletion struct {
	Deleted bool
	Lookup  *FirewallRecord
	Delete  *FirewallRecord
}

func firewallDeleteProfile(family string) neutronfind.Profile {
	p := neutronfind.Profile{Kind: family, Path: "fwaas/" + family, Plural: family, IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true, VersionedPath: "/v2.0/fwaas/" + family, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true}
	var body, query []string
	switch family {
	case "firewall_rules":
		p.Singular = "firewall_rule"
		body = []string{"id", "action", "description", "destination_ip_address", "destination_port", "enabled", "ip_version", "name", "project_id", "protocol", "shared", "source_ip_address", "source_port", "firewall_policy_id"}
		query = []string{"action", "description", "destination_ip_address", "name", "destination_port", "enabled", "fields", "ip_version", "project_id", "protocol", "shared", "source_ip_address", "source_port", "firewall_policy_id"}
	case "firewall_policies":
		p.Singular = "firewall_policy"
		body = []string{"id", "audited", "description", "firewall_rules", "name", "project_id", "shared"}
		query = []string{"description", "fields", "firewall_rules", "name", "project_id", "shared"}
	case "firewall_groups":
		p.Singular = "firewall_group"
		body = []string{"id", "admin_state_up", "description", "egress_firewall_policy_id", "ingress_firewall_policy_id", "name", "ports", "project_id", "shared", "status"}
		query = []string{"description", "egress_firewall_policy_id", "fields", "ingress_firewall_policy_id", "name", "shared", "status", "ports", "project_id"}
	}
	for _, key := range body {
		d := neutronfind.Descriptor{Name: key, Wire: key, Kind: neutronfind.Untyped, Missing: "null"}
		if key == "ports" || key == "firewall_rules" {
			d.Kind = neutronfind.List
		}
		p.Body = append(p.Body, d)
	}
	for _, key := range append([]string{"limit", "marker"}, query...) {
		p.Query = append(p.Query, neutronfind.QueryPair{Name: key, Wire: key})
	}
	return p
}
func firewallDeleteRecord(v *neutronfind.Record) *FirewallRecord {
	if v == nil {
		return nil
	}
	return &FirewallRecord{Resource: v.Resource.Clone(), Wire: v.Wire.Clone(), Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
}
func firewallDeleteProof(v *neutronfind.Record, err error) *FirewallRecord {
	if v != nil {
		return firewallDeleteRecord(v)
	}
	var accepted *resource.ResponseError
	if errors.As(err, &accepted) {
		return &FirewallRecord{Envelope: bytes.Clone(accepted.Body), Header: accepted.Header.Clone(), StatusCode: accepted.StatusCode}
	}
	receipt := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		receipt = auth.ErrOriginal
	}
	var rejected gophercloud.ErrUnexpectedResponseCode
	if errors.As(receipt, &rejected) {
		return &FirewallRecord{Envelope: bytes.Clone(rejected.Body), Header: rejected.ResponseHeader.Clone(), StatusCode: rejected.Actual}
	}
	return nil
}

// firewallDeleteMissing accepts only clean missing proof, never a joined
// callback/physical/context/source failure that happens to contain a 404.
func firewallDeleteMissing(err error) bool {
	if err == nil {
		return false
	}
	switch value := err.(type) {
	case *resource.ResponseError:
		return false
	case *resource.NotFoundError:
		if value.Cause == nil {
			return true
		}
		return firewallDeleteMissing(value.Cause)
	case gophercloud.ErrUnexpectedResponseCode:
		return value.Actual == http.StatusNotFound
	}
	if terminal, ok := err.(interface{ TerminalSDKFailure() bool }); ok && terminal.TerminalSDKFailure() {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		return len(causes) == 1 && firewallDeleteMissing(causes[0])
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return firewallDeleteMissing(wrapped.Unwrap())
	}
	return false
}

func firewallDeleteLocation(v *FirewallRecord, family string, location *resource.CloudLocation) error {
	if v == nil || v.Resource == nil {
		return nil
	}
	if family == "firewall_rules" {
		// Pinned _collect_attrs puts unknowns in Body, then replaces computed
		// input through the inherited empty _compute_attributes result. A
		// matching Rule therefore reads its Computed summary default, while
		// the actual response member remains unchanged in Wire.
		v.Resource.Body["summary"] = json.RawMessage(`""`)
	}
	if location == nil {
		v.Resource.Body["location"] = json.RawMessage("null")
		return nil
	}
	project := v.Resource.Body["project_id"]
	if project != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(project)
		if err != nil {
			return err
		}
		if !truthy {
			project = nil
		} else {
			current := location.Project.ID
			if current == nil {
				current = json.RawMessage("null")
			}
			equal, err := jsonfilter.EqualPythonLoadedJSON(project, current)
			if err != nil {
				return err
			}
			if equal {
				project = nil
			}
		}
	}
	raw, err := location.ForResource(project, v.Resource.Body["availability_zone"])
	if err != nil {
		return err
	}
	v.Resource.Body["location"] = bytes.Clone(raw)
	return nil
}

func (s *Service) deleteFirewallRecord(ctx context.Context, family, op, identity string, options []FirewallDeleteOption) (*FirewallDeletion, error) {
	result := &FirewallDeletion{}
	fail := func(err error) (*FirewallDeletion, error) {
		return result, request.Wrap(op, family, cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if s == nil || s.client == nil || s.API == nil || s.API.RawClient() != s.client {
		return fail(qosReadInvalid("configured Neutron service required"))
	}
	selected := s.API
	source, err := cloudread.Capture(ctx, s.client, "network")
	if err != nil {
		return fail(err)
	}
	guard := func(ctx context.Context) error {
		if s.API != selected || s.API.RawClient() != s.client {
			return qosReadInvalid("selected firewall service/API changed")
		}
		return source.Guard(ctx)
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	cfg := FirewallDeleteOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return fail(err)
		}
		if apply == nil {
			return fail(qosReadInvalid("nil firewall delete option"))
		}
		next := copyFirewallDelete(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return fail(err)
		}
		cfg = copyFirewallDelete(next)
	}
	members, err := qosPolicyCloudPairs(cfg.Filters, false)
	if err != nil {
		return fail(err)
	}
	strict := false
	policy := neutronfind.Options{IgnoreMissing: &strict, Query: make(map[string]json.RawMessage), Headers: cfg.Headers, Microversion: cfg.Microversion}
	for _, m := range members {
		if m.Key == "name_or_id" || m.Key == "ignore_missing" {
			return fail(qosReadInvalid("%q duplicates cloud Find argument", m.Key))
		}
		policy.Query[m.Key] = bytes.Clone(m.Value)
		policy.Order = append(policy.Order, m.Key)
	}
	engine, err := neutronfind.New(ctx, &source.Client, firewallDeleteProfile(family), guard)
	if err != nil {
		return fail(err)
	}
	var found *neutronfind.Record
	// Reuse Collection.Get's clean native404 classifier. This adapter delegates
	// the one strict Source Find; it never makes a second physical lookup.
	lookup := resource.NewCollection(resource.Adapter[neutronfind.Record]{Kind: family, ValidateID: func(string) error { return nil }, Get: func(context.Context, string) (*neutronfind.Record, error) { return engine.Find(ctx, identity, policy) }})
	found, err = lookup.Get(ctx, identity)
	result.Lookup = firewallDeleteProof(found, err)
	if err = errors.Join(err, guard(ctx)); err != nil {
		if firewallDeleteMissing(err) && guard(ctx) == nil {
			return result, nil
		}
		return fail(err)
	}
	if err = firewallDeleteLocation(result.Lookup, family, cfg.CurrentLocation); err != nil {
		if result.Lookup != nil {
			err = (&rest.Response{Body: bytes.Clone(result.Lookup.Envelope), Header: result.Lookup.Header.Clone(), StatusCode: result.Lookup.StatusCode}).Fail(err)
		}
		return fail(err)
	}
	if found == nil || found.Resource == nil {
		return fail(qosReadInvalid("strict firewall Find returned no resource"))
	}
	id := found.Resource.Body["id"]
	truthy, err := cloudfilter.PythonJSONTruthy(id)
	if err != nil || !truthy {
		return fail(errors.Join(err, qosReadInvalid("found firewall requires a nonempty member ID; collection DELETE is not enabled")))
	}
	member, err := cloudfilter.PythonString(id)
	if err != nil {
		return fail(err)
	}
	deletion := resource.NewCollection(resource.Adapter[neutronfind.Record]{Kind: family, ValidateID: func(string) error { return nil }, Delete: func(context.Context, string) error {
		value, cause := engine.Mutate(ctx, http.MethodDelete, member, nil, nil, neutronfind.Options{Headers: cfg.Headers, Microversion: cfg.Microversion})
		result.Delete = firewallDeleteProof(value, cause)
		return cause
	}})
	err = deletion.Delete(ctx, resource.ID(member), resource.WithMissingError())
	if err = errors.Join(err, guard(ctx)); err != nil {
		if firewallDeleteMissing(err) && guard(ctx) == nil {
			return result, nil
		}
		return fail(err)
	}
	result.Deleted = true
	return result, nil
}
func (s *Service) DeleteFirewallRuleRecord(ctx context.Context, identity string, options ...FirewallDeleteOption) (*FirewallDeletion, error) {
	return s.deleteFirewallRecord(ctx, "firewall_rules", "DeleteFirewallRuleRecord", identity, options)
}
func (s *Service) DeleteFirewallPolicyRecord(ctx context.Context, identity string, options ...FirewallDeleteOption) (*FirewallDeletion, error) {
	return s.deleteFirewallRecord(ctx, "firewall_policies", "DeleteFirewallPolicyRecord", identity, options)
}
func (s *Service) DeleteFirewallGroupRecord(ctx context.Context, identity string, options ...FirewallDeleteOption) (*FirewallDeletion, error) {
	return s.deleteFirewallRecord(ctx, "firewall_groups", "DeleteFirewallGroupRecord", identity, options)
}
