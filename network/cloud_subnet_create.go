package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SubnetCreateOpts owns Source optional arguments and arbitrary extension kwargs.
// Missing ip_version and enable_dhcp select 4 and false. Nil raw means null.
type SubnetCreateOpts struct {
	Fields          map[string]json.RawMessage
	CurrentLocation *resource.CloudLocation
	order           []string
}
type SubnetCreateOption func(*SubnetCreateOpts) error

func cloneSubnetCreate(v SubnetCreateOpts) SubnetCreateOpts {
	out := SubnetCreateOpts{Fields: make(map[string]json.RawMessage, len(v.Fields)), order: slices.Clone(v.order)}
	for key, raw := range v.Fields {
		out.Fields[key] = bytes.Clone(raw)
	}
	if v.CurrentLocation != nil {
		x := v.CurrentLocation.Clone()
		out.CurrentLocation = &x
	}
	return out
}
func WithSubnetCreateOpts(v SubnetCreateOpts) SubnetCreateOption {
	owned := cloneSubnetCreate(v)
	return func(c *SubnetCreateOpts) error {
		next := cloneSubnetCreate(owned)
		if next.CurrentLocation == nil && c.CurrentLocation != nil {
			x := c.CurrentLocation.Clone()
			next.CurrentLocation = &x
		}
		*c = next
		return nil
	}
}

// WithSubnetCreateAttribute binds a Source formal argument when its name is
// declared, otherwise it supplies an arbitrary Proxy/extension attribute.
func WithSubnetCreateAttribute(key string, value any) SubnetCreateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *SubnetCreateOpts) error {
		if err != nil {
			return qosReadInvalid("subnet keyword %q JSON: %v", key, err)
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		if _, present := c.Fields[key]; !present {
			c.order = append(c.order, key)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}

// WithSubnetCreateJSON replaces the complete Source keyword map and preserves
// JSON member order. Later individual options replace only their own keyword.
func WithSubnetCreateJSON(raw json.RawMessage) SubnetCreateOption {
	owned := bytes.Clone(raw)
	return func(c *SubnetCreateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("subnet keywords require complete UTF-8 JSON object: %v", err)
		}
		c.Fields = make(map[string]json.RawMessage, len(members))
		c.order = nil
		for _, member := range members {
			if _, present := c.Fields[member.Key]; !present {
				c.order = append(c.order, member.Key)
			}
			c.Fields[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithSubnetCreateCurrentLocation(v resource.CloudLocation) SubnetCreateOption {
	owned := v.Clone()
	return func(c *SubnetCreateOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}
func WithSubnetCreateCIDR(v string) SubnetCreateOption { return WithSubnetCreateAttribute("cidr", v) }
func WithSubnetCreateIPVersion(v any) SubnetCreateOption {
	return WithSubnetCreateAttribute("ip_version", v)
}
func WithSubnetCreateDHCP(v bool) SubnetCreateOption {
	return WithSubnetCreateAttribute("enable_dhcp", v)
}
func WithSubnetCreateName(v string) SubnetCreateOption {
	return WithSubnetCreateAttribute("subnet_name", v)
}
func WithSubnetCreateTenantID(v string) SubnetCreateOption {
	return WithSubnetCreateAttribute("tenant_id", v)
}
func WithSubnetCreateGatewayIP(v string) SubnetCreateOption {
	return WithSubnetCreateAttribute("gateway_ip", v)
}
func WithSubnetCreateDisableGateway(v bool) SubnetCreateOption {
	return WithSubnetCreateAttribute("disable_gateway_ip", v)
}
func WithSubnetCreateAllocationPools(v any) SubnetCreateOption {
	return WithSubnetCreateAttribute("allocation_pools", v)
}
func WithSubnetCreateDNSNameservers(v []string) SubnetCreateOption {
	return WithSubnetCreateAttribute("dns_nameservers", v)
}
func WithSubnetCreateHostRoutes(v any) SubnetCreateOption {
	return WithSubnetCreateAttribute("host_routes", v)
}
func WithSubnetCreateIPv6RAMode(v string) SubnetCreateOption {
	return WithSubnetCreateAttribute("ipv6_ra_mode", v)
}
func WithSubnetCreateIPv6AddressMode(v string) SubnetCreateOption {
	return WithSubnetCreateAttribute("ipv6_address_mode", v)
}
func WithSubnetCreatePrefixLength(v any) SubnetCreateOption {
	return WithSubnetCreateAttribute("prefixlen", v)
}
func WithSubnetCreateDefaultSubnetPool(v bool) SubnetCreateOption {
	return WithSubnetCreateAttribute("use_default_subnetpool", v)
}
func WithSubnetCreateSubnetPool(v string) SubnetCreateOption {
	return WithSubnetCreateAttribute("subnetpool_name_or_id", v)
}

// SubnetCreateResult separates lookup observations from the actual subnet POST.
// Failed response handling never invents Resource/Wire or a successful status.
type SubnetCreateResult struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Network        *networks.NetworkRecord
	SubnetPool     *subnetpools.SubnetPoolRecord
}

var subnetCreateFormals = map[string]bool{
	"cidr": true, "ip_version": true, "enable_dhcp": true, "subnet_name": true,
	"tenant_id": true, "allocation_pools": true, "gateway_ip": true, "disable_gateway_ip": true,
	"dns_nameservers": true, "host_routes": true, "ipv6_ra_mode": true, "ipv6_address_mode": true,
	"prefixlen": true, "use_default_subnetpool": true, "subnetpool_name_or_id": true,
}

func subnetCreateKeys(fields map[string]json.RawMessage, order []string) []string {
	keys := make([]string, 0, len(fields))
	seen := make(map[string]bool, len(fields))
	for _, key := range order {
		if _, present := fields[key]; present && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	return keys
}
func (s *Service) prepareSubnetCreate(ctx context.Context, options []SubnetCreateOption) (context.Context, *networkv2.Service, SubnetCreateOpts, func(context.Context) error, error) {
	if err := cloudread.Context(ctx); err != nil {
		return ctx, nil, SubnetCreateOpts{}, nil, err
	}
	if s == nil || s.client == nil || s.API == nil || s.API.RawClient() != s.client || s.API.Networks == nil || s.API.Subnets == nil || s.API.SubnetPools == nil {
		return ctx, nil, SubnetCreateOpts{}, nil, qosReadInvalid("configured network service and subnet dependencies required")
	}
	selectedAPI, selectedClient := s.API, s.client
	selectedNetwork, selectedSubnets, selectedPools := selectedAPI.Networks, selectedAPI.Subnets, selectedAPI.SubnetPools
	source, err := cloudread.Capture(ctx, selectedClient, "network")
	if err != nil {
		return ctx, nil, SubnetCreateOpts{}, nil, err
	}
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && (s.API != selectedAPI || s.client != selectedClient || selectedAPI.RawClient() != selectedClient || selectedAPI.Networks != selectedNetwork || selectedAPI.Subnets != selectedSubnets || selectedAPI.SubnetPools != selectedPools) {
			changed = qosReadInvalid("selected subnet service/API changed")
		}
		return errors.Join(changed, source.Guard(ctx))
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	cfg := cloneSubnetCreate(SubnetCreateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return ctx, nil, SubnetCreateOpts{}, guard, err
		}
		if apply == nil {
			return ctx, nil, SubnetCreateOpts{}, guard, qosReadInvalid("nil subnet create option")
		}
		next := cloneSubnetCreate(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return ctx, nil, SubnetCreateOpts{}, guard, err
		}
		cfg = cloneSubnetCreate(next)
	}
	for _, key := range subnetCreateKeys(cfg.Fields, cfg.order) {
		if key == "network_name_or_id" || key == "self" {
			return ctx, nil, SubnetCreateOpts{}, guard, qosReadInvalid("%q is already a positional/receiver argument", key)
		}
		if !utf8.ValidString(key) {
			return ctx, nil, SubnetCreateOpts{}, guard, qosReadInvalid("subnet keyword name must be UTF-8")
		}
		if cfg.Fields[key] == nil {
			cfg.Fields[key] = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(cfg.Fields[key]); err != nil {
			return ctx, nil, SubnetCreateOpts{}, guard, qosReadInvalid("subnet keyword %q requires complete UTF-8 JSON: %v", key, err)
		}
	}
	return ctx, networkv2.New(&source.Client), cfg, guard, guard(ctx)
}
func subnetCreateTruthy(fields map[string]json.RawMessage, key string) bool {
	raw, present := fields[key]
	if !present {
		return false
	}
	truth, _ := cloudfilter.PythonJSONTruthy(raw)
	return truth
}
func subnetCreateWrap(ctx context.Context, err error) error {
	return request.Wrap("CreateSubnetRecord", "subnets", cloudread.ContextError(ctx, err))
}

// CreateSubnetRecord owns the cloud lookup/validation/default/translation flow.
// Gateway/CIDR conflicts and string IP conversion occur after the Source lookups.
func (s *Service) CreateSubnetRecord(ctx context.Context, network string, options ...SubnetCreateOption) (*SubnetCreateResult, error) {
	ctx, api, cfg, guard, err := s.prepareSubnetCreate(ctx, options)
	if err != nil {
		return nil, subnetCreateWrap(ctx, err)
	}
	formals := map[string]json.RawMessage{"ip_version": json.RawMessage("4"), "enable_dhcp": json.RawMessage("false")}
	attrs := make(map[string]json.RawMessage)
	var order []string
	for _, key := range subnetCreateKeys(cfg.Fields, cfg.order) {
		if subnetCreateFormals[key] {
			formals[key] = bytes.Clone(cfg.Fields[key])
		} else {
			attrs[key] = bytes.Clone(cfg.Fields[key])
			order = append(order, key)
		}
	}
	result := &SubnetCreateResult{}
	result.Network, err = api.Networks.FindSubnetSourceRecord(ctx, network, formals["tenant_id"])
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, subnetCreateWrap(ctx, err)
	}
	if result.Network == nil || result.Network.Resource == nil {
		return result, subnetCreateWrap(ctx, &resource.NotFoundError{Resource: "network", Reference: network})
	}
	if cfg.CurrentLocation != nil {
		raw, cause := cloudNetworkReadLocation(result.Network.Resource.Body, cfg.CurrentLocation)
		if cause != nil {
			return result, subnetCreateWrap(ctx, cause)
		}
		result.Network.Resource.Body["location"] = raw
	}
	if subnetCreateTruthy(formals, "disable_gateway_ip") && subnetCreateTruthy(formals, "gateway_ip") {
		return result, subnetCreateWrap(ctx, qosReadInvalid("disable_gateway_ip is not allowed with gateway_ip"))
	}
	cidr := subnetCreateTruthy(formals, "cidr")
	defaultPool := subnetCreateTruthy(formals, "use_default_subnetpool")
	namedPool := subnetCreateTruthy(formals, "subnetpool_name_or_id")
	if !cidr && !defaultPool && !namedPool {
		return result, subnetCreateWrap(ctx, qosReadInvalid("cidr is required when a subnetpool is not used"))
	}
	if cidr && (defaultPool || namedPool) {
		return result, subnetCreateWrap(ctx, qosReadInvalid("cidr and subnetpool may not be used at the same time"))
	}
	if defaultPool && namedPool {
		return result, subnetCreateWrap(ctx, qosReadInvalid("use_default_subnetpool and subnetpool_name_or_id may not be used at the same time"))
	}
	if namedPool {
		var identity string
		if err := json.Unmarshal(formals["subnetpool_name_or_id"], &identity); err != nil {
			return result, subnetCreateWrap(ctx, qosReadInvalid("subnetpool_name_or_id requires a string reference"))
		}
		result.SubnetPool, err = api.SubnetPools.FindSubnetSourceRecord(ctx, identity)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return result, subnetCreateWrap(ctx, err)
		}
		if result.SubnetPool == nil || result.SubnetPool.Resource == nil {
			return result, subnetCreateWrap(ctx, &resource.NotFoundError{Resource: "subnetpool", Reference: identity})
		}
		if cfg.CurrentLocation != nil {
			raw, cause := cloudNetworkReadLocation(result.SubnetPool.Resource.Body, cfg.CurrentLocation)
			if cause != nil {
				return result, subnetCreateWrap(ctx, cause)
			}
			result.SubnetPool.Resource.Body["location"] = raw
		}
	}
	ip := formals["ip_version"]
	if len(bytes.TrimSpace(ip)) > 0 && bytes.TrimSpace(ip)[0] == '"' {
		var text string
		_ = json.Unmarshal(ip, &text)
		value, cause := jsonfilter.PythonIntegerString(text)
		if cause != nil {
			return result, subnetCreateWrap(ctx, qosReadInvalid("ip_version must be an integer: %v", cause))
		}
		ip = json.RawMessage(value.String())
	}
	body := make(map[string]json.RawMessage, len(attrs)+16)
	bodyOrder := []string{"network_id", "ip_version", "enable_dhcp"}
	body["network_id"] = bytes.Clone(result.Network.Resource.Body["id"])
	body["ip_version"] = bytes.Clone(ip)
	body["enable_dhcp"] = bytes.Clone(formals["enable_dhcp"])
	put := func(key string, raw json.RawMessage) {
		if _, exists := body[key]; !exists {
			bodyOrder = append(bodyOrder, key)
		}
		body[key] = bytes.Clone(raw)
	}
	for _, key := range order {
		put(key, attrs[key])
	}
	for _, pair := range [][2]string{{"cidr", "cidr"}, {"subnet_name", "name"}, {"tenant_id", "tenant_id"}, {"allocation_pools", "allocation_pools"}, {"gateway_ip", "gateway_ip"}} {
		if subnetCreateTruthy(formals, pair[0]) {
			put(pair[1], formals[pair[0]])
		}
	}
	if subnetCreateTruthy(formals, "disable_gateway_ip") {
		put("gateway_ip", json.RawMessage("null"))
	}
	for _, pair := range [][2]string{{"dns_nameservers", "dns_nameservers"}, {"host_routes", "host_routes"}, {"ipv6_ra_mode", "ipv6_ra_mode"}, {"ipv6_address_mode", "ipv6_address_mode"}, {"prefixlen", "prefixlen"}} {
		if subnetCreateTruthy(formals, pair[0]) {
			put(pair[1], formals[pair[0]])
		}
	}
	if defaultPool {
		put("use_default_subnetpool", json.RawMessage("true"))
	}
	if result.SubnetPool != nil {
		put("subnetpool_id", result.SubnetPool.Resource.Body["id"])
	}
	value, err := api.Subnets.CreateCloudRecord(ctx, body, subnets.WithCloudCreateOrder(bodyOrder...))
	if value != nil {
		result.Resource, result.Wire, result.Envelope, result.Header, result.StatusCode = value.Resource, value.Wire, value.Envelope, value.Header, value.StatusCode
		// Invalidate the private successful topology snapshot after an actual
		// accepted mutation, even when later body/Close/projection work failed.
		if value.StatusCode >= 100 && value.StatusCode < 400 && s.mutationRoles != nil {
			s.mutationRoles.Reset()
		}
	}
	if result.Resource != nil && cfg.CurrentLocation != nil {
		raw, cause := cloudNetworkReadLocation(result.Resource.Body, cfg.CurrentLocation)
		err = errors.Join(err, cause)
		if cause == nil {
			result.Resource.Body["location"] = raw
		}
	}
	finalErr := errors.Join(err, guard(ctx))
	if finalErr != nil && result.StatusCode >= 100 && result.StatusCode < 400 {
		result.Resource = nil
		finalErr = &resource.ResponseError{StatusCode: result.StatusCode, Header: result.Header.Clone(), Body: bytes.Clone(result.Envelope), Cause: finalErr}
	}
	return result, subnetCreateWrap(ctx, finalErr)
}
