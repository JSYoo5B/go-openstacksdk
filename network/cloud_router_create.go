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
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RouterCreateOpts owns the seven pinned cloud kwargs. Omitted admin_state_up
// defaults to true. CurrentLocation is SDK metadata and never enters the POST.
type RouterCreateOpts struct {
	Attributes      map[string]json.RawMessage
	CurrentLocation *resource.CloudLocation
}
type RouterCreateOption func(*RouterCreateOpts) error

func copyRouterCreate(v RouterCreateOpts) RouterCreateOpts {
	out := RouterCreateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes))}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	if v.CurrentLocation != nil {
		x := v.CurrentLocation.Clone()
		out.CurrentLocation = &x
	}
	return out
}
func WithRouterCreateOpts(v RouterCreateOpts) RouterCreateOption {
	owned := copyRouterCreate(v)
	return func(c *RouterCreateOpts) error { *c = copyRouterCreate(owned); return nil }
}
func routerCreateKey(key string) bool {
	switch key {
	case "name", "admin_state_up", "ext_gateway_net_id", "enable_snat", "ext_fixed_ips", "project_id", "availability_zone_hints":
		return true
	}
	return false
}
func WithRouterCreateAttribute(key string, v any) RouterCreateOption {
	if !routerCreateKey(key) {
		return func(*RouterCreateOpts) error { return qosReadInvalid("%q is not a pinned cloud router keyword", key) }
	}
	raw, err := json.Marshal(v)
	owned := bytes.Clone(raw)
	return func(c *RouterCreateOpts) error {
		if err != nil {
			return qosReadInvalid("attribute %q JSON: %v", key, err)
		}
		if c.Attributes == nil {
			c.Attributes = make(map[string]json.RawMessage)
		}
		c.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}

// WithRouterCreateJSON replaces kwargs using an owned complete JSON object.
// It exposes Source null/false/non-string forwarding without custom builders.
func WithRouterCreateJSON(v json.RawMessage) RouterCreateOption {
	owned := bytes.Clone(v)
	return func(c *RouterCreateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("router kwargs require complete UTF-8 JSON object: %v", err)
		}
		c.Attributes = make(map[string]json.RawMessage, len(members))
		for _, member := range members {
			c.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithRouterCreateName(v string) RouterCreateOption { return WithRouterCreateAttribute("name", v) }
func WithRouterCreateAdminStateUp(v bool) RouterCreateOption {
	return WithRouterCreateAttribute("admin_state_up", v)
}
func WithRouterCreateExternalGatewayNetworkID(v string) RouterCreateOption {
	return WithRouterCreateAttribute("ext_gateway_net_id", v)
}
func WithRouterCreateEnableSNAT(v bool) RouterCreateOption {
	return WithRouterCreateAttribute("enable_snat", v)
}
func WithRouterCreateExternalFixedIPs(v []map[string]any) RouterCreateOption {
	return WithRouterCreateAttribute("ext_fixed_ips", v)
}
func WithRouterCreateProjectID(v string) RouterCreateOption {
	return WithRouterCreateAttribute("project_id", v)
}
func WithRouterCreateAvailabilityZoneHints(v ...string) RouterCreateOption {
	// No arguments is an explicit empty list, distinct from omitted/null hints.
	return WithRouterCreateAttribute("availability_zone_hints", append([]string{}, v...))
}
func WithRouterCreateCurrentLocation(v resource.CloudLocation) RouterCreateOption {
	owned := v.Clone()
	return func(c *RouterCreateOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}

// RouterCreateResult separates the SDK Resource view, actual router response,
// whole HTTP receipt and independent extension discovery. A gate failure cannot
// fabricate a POST receipt; rejected/read/decode failure cannot fabricate Router.
type RouterCreateResult struct {
	Router, Wire *resource.RawResource
	Envelope     json.RawMessage
	Header       http.Header
	StatusCode   int
	Extensions   []*extensions.ExtensionRecord
}

func routerCreateBody(cfg RouterCreateOpts) (map[string]json.RawMessage, bool, error) {
	attrs := copyRouterCreate(cfg).Attributes
	for _, key := range slices.Sorted(maps.Keys(attrs)) {
		if !routerCreateKey(key) {
			return nil, false, qosReadInvalid("%q is not a pinned cloud router keyword", key)
		}
		raw := attrs[key]
		if raw == nil {
			raw = json.RawMessage("null")
			attrs[key] = raw
		}
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return nil, false, qosReadInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err)
		}
	}
	body := map[string]json.RawMessage{"admin_state_up": json.RawMessage("true")}
	if raw, present := attrs["admin_state_up"]; present {
		body["admin_state_up"] = bytes.Clone(raw)
	}
	nonnull := func(raw json.RawMessage) bool {
		return raw != nil && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
	}
	if raw := attrs["project_id"]; nonnull(raw) {
		body["project_id"] = bytes.Clone(raw)
	}
	truthy := func(raw json.RawMessage) bool {
		if raw == nil {
			return false
		}
		truth, _ := cloudfilter.PythonJSONTruthy(raw)
		return truth
	}
	if raw := attrs["name"]; truthy(raw) {
		body["name"] = bytes.Clone(raw)
	}
	gateway := map[string]json.RawMessage{}
	if raw := attrs["ext_gateway_net_id"]; truthy(raw) {
		gateway["network_id"] = bytes.Clone(raw)
	}
	if raw := attrs["enable_snat"]; nonnull(raw) {
		gateway["enable_snat"] = bytes.Clone(raw)
	}
	if raw := attrs["ext_fixed_ips"]; truthy(raw) {
		gateway["external_fixed_ips"] = bytes.Clone(raw)
	}
	if len(gateway) > 0 {
		raw, err := json.Marshal(gateway)
		if err != nil {
			return nil, false, err
		}
		body["external_gateway_info"] = raw
	}
	hints := attrs["availability_zone_hints"]
	if nonnull(hints) {
		if bytes.TrimSpace(hints)[0] != '[' {
			return nil, false, qosReadInvalid("Parameter 'availability_zone_hints' must be a list")
		}
		body["availability_zone_hints"] = bytes.Clone(hints)
		return body, true, nil
	}
	return body, false, nil
}

// CreateRouterRecord follows pinned create_router defaults, gateway assembly and
// conditional fresh complete AZ discovery before one collection POST. Server
// policy/range/driver validation remains server-owned. Native APIs are intact.
func (s *Service) CreateRouterRecord(ctx context.Context, options ...RouterCreateOption) (*RouterCreateResult, error) {
	const op = "CreateRouterRecord"
	wrap := func(e error) error { return request.Wrap(op, "routers", cloudread.ContextError(ctx, e)) }
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
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && (s.API != selectedAPI || s.client != selectedClient || selectedAPI.RawClient() != selectedClient) {
			observed = qosReadInvalid("selected network service/API changed")
		}
		return errors.Join(observed, source.Guard(ctx))
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	cfg := copyRouterCreate(RouterCreateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil cloud router create option"))
		}
		next := copyRouterCreate(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyRouterCreate(next)
	}
	body, hasHints, err := routerCreateBody(cfg)
	if err != nil {
		return nil, wrap(err)
	}
	if cfg.CurrentLocation != nil {
		if _, err := cloudNetworkReadLocation(body, cfg.CurrentLocation); err != nil {
			return nil, wrap(err)
		}
	}
	if err := guard(ctx); err != nil {
		return nil, wrap(err)
	}
	selected := &Service{client: &source.Client, API: networkv2.New(&source.Client)}
	result := &RouterCreateResult{}
	if hasHints {
		state, err := qosReadStart(ctx, selected, "router_availability_zone")
		if state != nil {
			result.Extensions = append(result.Extensions, state.Extensions...)
		}
		if err != nil {
			return result, wrap(errors.Join(err, guard(ctx)))
		}
		ctx = rest.WithOperationGuard(ctx, state.Check)
	}
	value, err := selected.API.Routers.CreateCloudRecord(ctx, body)
	if value != nil {
		result.Router, result.Wire, result.Envelope, result.Header, result.StatusCode = value.Resource, value.Wire, bytes.Clone(value.Envelope), value.Header.Clone(), value.StatusCode
		if result.Router != nil && cfg.CurrentLocation != nil {
			raw, cause := cloudNetworkReadLocation(result.Router.Body, cfg.CurrentLocation)
			if cause == nil {
				result.Router.Body["location"] = bytes.Clone(raw)
			} else {
				response := &rest.Response{Body: bytes.Clone(result.Envelope), Header: result.Header.Clone(), StatusCode: result.StatusCode}
				err = errors.Join(err, response.Fail(cause))
				result.Router = nil
			}
		}
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
