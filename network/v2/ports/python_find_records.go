package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PortRecord owns the declared Source view and original response evidence.
type PortRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
	source         neutronfind.SourceState
}

func (v *PortRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	v.row = bytes.Clone(data)
	return json.Unmarshal(data, v.Wire)
}
func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// recordFindProfile preserves this resource's pinned Body/Query declarations.
// NetworkResource supplies only the explicitly declared Header profile below.
var recordFindProfile = neutronfind.Profile{
	Kind: "ports", Path: "ports", Singular: "port", Plural: "ports",
	IDKey: "id", NameKey: "name",
	PreserveUnknownBody: true, NameQueryHint: true, VersionedPath: "/v2.0/ports",
	Headers: []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Kind: neutronfind.List, Missing: "null"}},
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "revision_number", Wire: "revision_number", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "tags", Wire: "tags", Kind: neutronfind.List, Missing: "[]"},
		{Name: "allowed_address_pairs", Wire: "allowed_address_pairs", Kind: neutronfind.List, Missing: "null"},
		{Name: "binding_host_id", Wire: "binding:host_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "binding_profile", Wire: "binding:profile", Kind: neutronfind.Dictionary, Missing: "null"},
		{Name: "binding_vif_details", Wire: "binding:vif_details", Kind: neutronfind.Dictionary, Missing: "null"},
		{Name: "binding_vif_type", Wire: "binding:vif_type", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "binding_vnic_type", Wire: "binding:vnic_type", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "created_at", Wire: "created_at", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "data_plane_status", Wire: "data_plane_status", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "description", Wire: "description", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "device_id", Wire: "device_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "device_owner", Wire: "device_owner", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "device_profile", Wire: "device_profile", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "dns_assignment", Wire: "dns_assignment", Kind: neutronfind.List, Missing: "null"},
		{Name: "dns_domain", Wire: "dns_domain", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "dns_name", Wire: "dns_name", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "extra_dhcp_opts", Wire: "extra_dhcp_opts", Kind: neutronfind.List, Missing: "null"},
		{Name: "fixed_ips", Wire: "fixed_ips", Kind: neutronfind.List, Missing: "null"},
		{Name: "hardware_offload_type", Wire: "hardware_offload_type", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "ip_allocation", Wire: "ip_allocation", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "is_admin_state_up", Wire: "admin_state_up", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "is_port_security_enabled", Wire: "port_security_enabled", Kind: neutronfind.Boolean, Missing: "false"},
		{Name: "mac_address", Wire: "mac_address", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "name", Wire: "name", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "network_id", Wire: "network_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "numa_affinity_policy", Wire: "numa_affinity_policy", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "project_id", Wire: "project_id", Kind: neutronfind.Untyped, Missing: "null", Alias: "tenant_id"},
		{Name: "tenant_id", Wire: "tenant_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "propagate_uplink_status", Wire: "propagate_uplink_status", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "pvlan_type", Wire: "pvlan_type", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "pvlan_community", Wire: "pvlan_community", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "qos_network_policy_id", Wire: "qos_network_policy_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "qos_policy_id", Wire: "qos_policy_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "resource_request", Wire: "resource_request", Kind: neutronfind.Dictionary, Missing: "null"},
		{Name: "security_group_ids", Wire: "security_groups", Kind: neutronfind.List, Missing: "null"},
		{Name: "status", Wire: "status", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "trunk_details", Wire: "trunk_details", Kind: neutronfind.Dictionary, Missing: "null"},
		{Name: "trusted", Wire: "trusted", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "updated_at", Wire: "updated_at", Kind: neutronfind.Untyped, Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"},
		{Name: "marker", Wire: "marker"},
		{Name: "binding:host_id", Wire: "binding:host_id"},
		{Name: "binding:profile", Wire: "binding:profile"},
		{Name: "binding:vif_details", Wire: "binding:vif_details"},
		{Name: "binding:vif_type", Wire: "binding:vif_type"},
		{Name: "binding:vnic_type", Wire: "binding:vnic_type"},
		{Name: "description", Wire: "description"},
		{Name: "device_id", Wire: "device_id"},
		{Name: "device_owner", Wire: "device_owner"},
		{Name: "fields", Wire: "fields"},
		{Name: "fixed_ips", Wire: "fixed_ips"},
		{Name: "id", Wire: "id"},
		{Name: "ip_address", Wire: "ip_address"},
		{Name: "mac_address", Wire: "mac_address"},
		{Name: "name", Wire: "name"},
		{Name: "network_id", Wire: "network_id"},
		{Name: "status", Wire: "status"},
		{Name: "subnet_id", Wire: "subnet_id"},
		{Name: "project_id", Wire: "project_id"},
		{Name: "security_groups", Wire: "security_groups"},
		{Name: "sort_key", Wire: "sort_key"},
		{Name: "sort_dir", Wire: "sort_dir"},
		{Name: "tenant_id", Wire: "project_id"},
		{Name: "is_admin_state_up", Wire: "admin_state_up"},
		{Name: "is_port_security_enabled", Wire: "port_security_enabled"},
		{Name: "security_group_ids", Wire: "security_groups"},
		{Name: "tags", Wire: "tags"},
		{Name: "any_tags", Wire: "tags-any"},
		{Name: "not_tags", Wire: "not-tags"},
		{Name: "not_any_tags", Wire: "not-tags-any"},
	},
}

// RecordFindOpts owns Source kwargs. Omitted IgnoreMissing defaults to true.
// Query preserves JSON values, including arrays and null; unknown kwargs go to
// the member GET but are ignored by the list, following Source Resource.find.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	queryOrder    []string
	Query         map[string]json.RawMessage
	Headers       map[string]string
}
type RecordFindOption func(*RecordFindOpts) error

func internalFindOpts(value RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: value.IgnoreMissing, Microversion: value.Microversion, Query: value.Query, Headers: value.Headers, Order: value.queryOrder}
}
func copyFindOpts(value RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(internalFindOpts(value))
	value.IgnoreMissing, value.Microversion = owned.IgnoreMissing, owned.Microversion
	value.Query, value.Headers, value.queryOrder = owned.Query, owned.Headers, owned.Order
	return value
}

func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyFindOpts(value)
	return func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil }
}
func WithRecordFindIgnoreMissing(value bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := value; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.queryOrder, key) {
			c.queryOrder = append(c.queryOrder, key)
		}
		return nil
	}
}
func WithRecordFindMicroversion(value string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := value; c.Microversion = &x; return nil }
}
func WithRecordFindHeaders(value map[string]string) RecordFindOption {
	owned := maps.Clone(value)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// FindRecord matches the pinned Source GET-first, complete ID-or-name search.
// It suppresses only clean GET 400/403/404. A physical body/Close/source/cancel
// failure never becomes a fallback or missing-resource success.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*PortRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "ports", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("ports API is required"))
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, recordFindProfile, func(ctx context.Context) error {
		if a.client != selected {
			return findInvalid("selected ports client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	guard := engine.Check
	policy := RecordFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil find option"))
		}
		next := copyFindOpts(policy)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	value, err := engine.Find(ctx, identity, internalFindOpts(policy))
	if value == nil {
		return nil, wrap(err)
	}
	result := &PortRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList(), source: value.Source()}
	return result, wrap(err)
}
