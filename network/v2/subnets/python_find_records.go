package subnets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SubnetRecord owns the declared Source view and original response evidence.
type SubnetRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
	source         neutronfind.SourceState
}

func (v *SubnetRecord) UnmarshalJSON(data []byte) error {
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
	Kind: "subnets", Path: "subnets", Singular: "subnet", Plural: "subnets",
	IDKey: "id", NameKey: "name",
	PreserveUnknownBody: true, NameQueryHint: true, VersionedPath: "/v2.0/subnets",
	Headers: []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Kind: neutronfind.List, Missing: "null"}},
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "revision_number", Wire: "revision_number", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "tags", Wire: "tags", Kind: neutronfind.List, Missing: "[]"},
		{Name: "allocation_pools", Wire: "allocation_pools", Kind: neutronfind.List, Missing: "null"},
		{Name: "cidr", Wire: "cidr", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "created_at", Wire: "created_at", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "description", Wire: "description", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "dns_nameservers", Wire: "dns_nameservers", Kind: neutronfind.List, Missing: "null"},
		{Name: "dns_publish_fixed_ip", Wire: "dns_publish_fixed_ip", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "gateway_ip", Wire: "gateway_ip", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "host_routes", Wire: "host_routes", Kind: neutronfind.List, Missing: "null"},
		{Name: "ip_version", Wire: "ip_version", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "ipv6_address_mode", Wire: "ipv6_address_mode", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "ipv6_ra_mode", Wire: "ipv6_ra_mode", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "is_dhcp_enabled", Wire: "enable_dhcp", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "name", Wire: "name", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "network_id", Wire: "network_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "prefix_length", Wire: "prefixlen", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "project_id", Wire: "project_id", Kind: neutronfind.Untyped, Missing: "null", Alias: "tenant_id"},
		{Name: "tenant_id", Wire: "tenant_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "segment_id", Wire: "segment_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "service_types", Wire: "service_types", Kind: neutronfind.List, Missing: "null"},
		{Name: "subnet_pool_id", Wire: "subnetpool_id", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "updated_at", Wire: "updated_at", Kind: neutronfind.Untyped, Missing: "null"},
		{Name: "use_default_subnet_pool", Wire: "use_default_subnetpool", Kind: neutronfind.Boolean, Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"},
		{Name: "marker", Wire: "marker"},
		{Name: "cidr", Wire: "cidr"},
		{Name: "description", Wire: "description"},
		{Name: "fields", Wire: "fields"},
		{Name: "gateway_ip", Wire: "gateway_ip"},
		{Name: "id", Wire: "id"},
		{Name: "ip_version", Wire: "ip_version"},
		{Name: "ipv6_address_mode", Wire: "ipv6_address_mode"},
		{Name: "ipv6_ra_mode", Wire: "ipv6_ra_mode"},
		{Name: "name", Wire: "name"},
		{Name: "network_id", Wire: "network_id"},
		{Name: "segment_id", Wire: "segment_id"},
		{Name: "dns_publish_fixed_ip", Wire: "dns_publish_fixed_ip"},
		{Name: "project_id", Wire: "project_id"},
		{Name: "sort_key", Wire: "sort_key"},
		{Name: "sort_dir", Wire: "sort_dir"},
		{Name: "is_dhcp_enabled", Wire: "enable_dhcp"},
		{Name: "subnet_pool_id", Wire: "subnetpool_id"},
		{Name: "use_default_subnet_pool", Wire: "use_default_subnetpool"},
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

// RecordFindOption is one option of FindRecord: WithRecordFindOpts,
// request.IgnoreMissing, request.Query (one Python kwarg as JSON; its marshal
// error is returned when applied), request.Microversion, request.Headers
// (replaces the headers) or request.Custom over *RecordFindOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type RecordFindOption = request.FindCall

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
	return request.Custom(func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil })
}

// recordFindCalls interprets the shared calls of FindRecord.
var recordFindCalls = calls.Spec[RecordFindOpts]{
	IgnoreMissing: calls.Pointer(func(c *RecordFindOpts) **bool { return &c.IgnoreMissing }),
	Query: func(c *RecordFindOpts, entry request.CallEntry) error {
		if entry.JSONErr != nil {
			return entry.JSONErr
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[entry.Key] = bytes.Clone(entry.JSON)
		if !slices.Contains(c.queryOrder, entry.Key) {
			c.queryOrder = append(c.queryOrder, entry.Key)
		}
		return nil
	},
	Microversion: calls.Pointer(func(c *RecordFindOpts) **string { return &c.Microversion }),
	Headers:      calls.Value(func(c *RecordFindOpts) *map[string]string { return &c.Headers }),
}

// FindRecord matches the pinned Source GET-first, complete ID-or-name search.
// It suppresses only clean GET 400/403/404. A physical body/Close/source/cancel
// failure never becomes a fallback or missing-resource success.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*SubnetRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "subnets", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("subnets API is required"))
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, recordFindProfile, func(ctx context.Context) error {
		if a.client != selected {
			return findInvalid("selected subnets client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	guard := engine.Check
	policy := RecordFindOpts{}
	for _, apply := range calls.Translate(options, recordFindCalls) {
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
	result := &SubnetRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList(), source: value.Source()}
	return result, wrap(err)
}
