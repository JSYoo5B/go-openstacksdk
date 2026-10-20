package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SubnetUpdateOpts owns the seven cloud update_subnet formal arguments.
// Only enable_dhcp uses nonnull selection; the other arguments use truthiness.
type SubnetUpdateOpts struct{ Attributes map[string]json.RawMessage }
type SubnetUpdateOption func(*SubnetUpdateOpts) error

func copySubnetUpdate(v SubnetUpdateOpts) SubnetUpdateOpts {
	out := SubnetUpdateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes))}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func subnetUpdateKey(key string) bool {
	switch key {
	case "subnet_name", "enable_dhcp", "gateway_ip", "disable_gateway_ip", "allocation_pools", "dns_nameservers", "host_routes":
		return true
	}
	return false
}
func WithSubnetUpdateOpts(v SubnetUpdateOpts) SubnetUpdateOption {
	owned := copySubnetUpdate(v)
	return func(c *SubnetUpdateOpts) error { *c = copySubnetUpdate(owned); return nil }
}
func WithSubnetUpdateAttribute(key string, value any) SubnetUpdateOption {
	if !subnetUpdateKey(key) {
		return func(*SubnetUpdateOpts) error {
			return qosReadInvalid("%q is not a pinned cloud subnet update keyword", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *SubnetUpdateOpts) error {
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
func WithSubnetUpdateJSON(raw json.RawMessage) SubnetUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *SubnetUpdateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("subnet kwargs require a complete UTF-8 JSON object: %v", err)
		}
		c.Attributes = map[string]json.RawMessage{}
		for _, member := range members {
			c.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithSubnetUpdateName(v string) SubnetUpdateOption {
	return WithSubnetUpdateAttribute("subnet_name", v)
}
func WithSubnetUpdateEnableDHCP(v bool) SubnetUpdateOption {
	return WithSubnetUpdateAttribute("enable_dhcp", v)
}
func WithSubnetUpdateGatewayIP(v string) SubnetUpdateOption {
	return WithSubnetUpdateAttribute("gateway_ip", v)
}
func WithSubnetUpdateDisableGatewayIP(v bool) SubnetUpdateOption {
	return WithSubnetUpdateAttribute("disable_gateway_ip", v)
}
func WithSubnetUpdateDNSNameservers(v ...string) SubnetUpdateOption {
	return WithSubnetUpdateAttribute("dns_nameservers", append([]string{}, v...))
}

func subnetUpdateBody(cfg SubnetUpdateOpts) (map[string]json.RawMessage, error) {
	body := map[string]json.RawMessage{}
	truth := func(key string) bool {
		v := cfg.Attributes[key]
		if v == nil {
			return false
		}
		t, _ := cloudfilter.PythonJSONTruthy(v)
		return t
	}
	for _, key := range []string{"subnet_name", "gateway_ip", "allocation_pools", "dns_nameservers", "host_routes"} {
		if truth(key) {
			wire := key
			if key == "subnet_name" {
				wire = "name"
			}
			body[wire] = bytes.Clone(cfg.Attributes[key])
		}
	}
	if raw := cfg.Attributes["enable_dhcp"]; raw != nil && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		body["enable_dhcp"] = bytes.Clone(raw)
	}
	if truth("disable_gateway_ip") {
		body["gateway_ip"] = json.RawMessage("null")
	}
	if len(body) == 0 {
		return body, nil
	}
	if truth("disable_gateway_ip") && truth("gateway_ip") {
		return nil, qosReadInvalid("arg:disable_gateway_ip is not allowed with arg:gateway_ip")
	}
	return body, nil
}

// SubnetUpdateResult retains the lookup's declared view and wire evidence. Subnet is
// the native PUT response only; lookup-only/no-dirty branches never fabricate it.
type SubnetUpdateResult struct {
	Lookup *subnets.SubnetRecord
	Subnet *subnets.Subnet
	NoOp   bool
}

// UpdateSubnetRecord skips all HTTP for ineffective kwargs, then compares
// supplied values to the looked-up view and commits only changed attributes.
// SDK-owned options replace Python default kwargs without application builders.
func (s *Service) UpdateSubnetRecord(ctx context.Context, identity string, options ...SubnetUpdateOption) (*SubnetUpdateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateSubnetRecord", "subnets", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	unchanged := func() bool { return s == nil }
	if s != nil {
		originalAPI, originalClient := s.API, s.client
		unchanged = func() bool { return s.API == originalAPI && s.client == originalClient }
	}
	cfg := copySubnetUpdate(SubnetUpdateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := cloudread.Context(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil cloud subnet update option"))
		}
		next := copySubnetUpdate(cfg)
		if err := errors.Join(apply(&next), cloudread.Context(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copySubnetUpdate(next)
	}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		if !subnetUpdateKey(key) {
			return nil, wrap(qosReadInvalid("%q is not a pinned cloud subnet update keyword", key))
		}
		if cfg.Attributes[key] == nil {
			cfg.Attributes[key] = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(cfg.Attributes[key]); err != nil {
			return nil, wrap(qosReadInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err))
		}
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	body, err := subnetUpdateBody(cfg)
	if err != nil {
		return nil, wrap(err)
	}
	if len(body) == 0 {
		return &SubnetUpdateResult{NoOp: true}, nil
	}
	if !unchanged() {
		return nil, wrap(qosReadInvalid("selected network service/API changed during options"))
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
	selected := networkv2.New(&source.Client)
	result := &SubnetUpdateResult{}
	result.Lookup, err = selected.Subnets.FindRecord(ctx, identity)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if result.Lookup == nil {
		return result, wrap(&resource.NotFoundError{Resource: "subnet", Reference: identity})
	}
	if result.Lookup.Resource == nil || result.Lookup.Wire == nil {
		return result, wrap(qosReadInvalid("subnet lookup did not produce a resource view"))
	}
	attrs := []subnets.AttributeOption{}
	for _, key := range slices.Sorted(maps.Keys(body)) {
		raw := body[key]
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
		attrs = append(attrs, subnets.WithAttribute(key, json.RawMessage(bytes.Clone(raw))))
	}
	if len(attrs) == 0 {
		result.NoOp = true
		return result, wrap(guard(ctx))
	}
	var id string
	if result.Lookup.Resource == nil || json.Unmarshal(result.Lookup.Resource.Body["id"], &id) != nil || id == "" {
		return result, wrap((&rest.Response{Body: result.Lookup.Envelope, Header: result.Lookup.Header, StatusCode: result.Lookup.StatusCode}).Fail(qosReadInvalid("subnet update requires a nonempty string response ID")))
	}
	if err := guard(ctx); err != nil {
		return result, wrap(err)
	}
	result.Subnet, err = selected.Subnets.UpdateWithAttributes(ctx, id, attrs...)
	if err != nil {
		result.Subnet = nil // A rejected PUT is not a successful native result.
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
