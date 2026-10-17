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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RouterUpdateOpts owns the six pinned cloud update_router formal arguments.
type RouterUpdateOpts struct{ Attributes map[string]json.RawMessage }
type RouterUpdateOption func(*RouterUpdateOpts) error

func copyRouterUpdate(v RouterUpdateOpts) RouterUpdateOpts {
	out := RouterUpdateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes))}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func routerUpdateKey(key string) bool {
	switch key {
	case "name", "admin_state_up", "ext_gateway_net_id", "enable_snat", "ext_fixed_ips", "routes":
		return true
	}
	return false
}
func WithRouterUpdateOpts(v RouterUpdateOpts) RouterUpdateOption {
	owned := copyRouterUpdate(v)
	return func(c *RouterUpdateOpts) error { *c = copyRouterUpdate(owned); return nil }
}
func WithRouterUpdateAttribute(key string, value any) RouterUpdateOption {
	if !routerUpdateKey(key) {
		return func(*RouterUpdateOpts) error {
			return qosReadInvalid("%q is not a pinned cloud router update keyword", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RouterUpdateOpts) error {
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
func WithRouterUpdateJSON(raw json.RawMessage) RouterUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *RouterUpdateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("router kwargs require a complete UTF-8 JSON object: %v", err)
		}
		c.Attributes = map[string]json.RawMessage{}
		for _, member := range members {
			c.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithRouterUpdateName(v string) RouterUpdateOption { return WithRouterUpdateAttribute("name", v) }
func WithRouterUpdateAdminStateUp(v bool) RouterUpdateOption {
	return WithRouterUpdateAttribute("admin_state_up", v)
}
func WithRouterUpdateExternalGatewayNetworkID(v string) RouterUpdateOption {
	return WithRouterUpdateAttribute("ext_gateway_net_id", v)
}
func WithRouterUpdateEnableSNAT(v bool) RouterUpdateOption {
	return WithRouterUpdateAttribute("enable_snat", v)
}
func WithRouterUpdateRoutes(v []map[string]any) RouterUpdateOption {
	return WithRouterUpdateAttribute("routes", v)
}
func routerUpdateBody(cfg RouterUpdateOpts) (map[string]json.RawMessage, bool, error) {
	attrs := copyRouterUpdate(cfg).Attributes
	routes, hasRoutes := attrs["routes"]
	hasRoutes = hasRoutes && !bytes.Equal(bytes.TrimSpace(routes), []byte("null"))
	delete(attrs, "routes")
	body, _, err := routerCreateBody(RouterCreateOpts{Attributes: attrs})
	if raw := attrs["admin_state_up"]; raw == nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		delete(body, "admin_state_up")
	}
	return body, hasRoutes, err
}

// RouterUpdateResult retains the lookup's declared view and wire evidence. Router is
// the native PUT response only; lookup-only/no-dirty branches never fabricate it.
type RouterUpdateResult struct {
	Lookup        *routers.RouterReadRecord
	Router        *routers.Router
	NoOp          bool
	RoutesDropped bool
	Extensions    []*extensions.ExtensionRecord
}

// UpdateRouterRecord skips all HTTP for ineffective kwargs, then compares
// supplied values to the looked-up view and commits only changed attributes.
// SDK-owned options replace Python default kwargs without application builders.
func (s *Service) UpdateRouterRecord(ctx context.Context, identity string, options ...RouterUpdateOption) (*RouterUpdateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateRouterRecord", "routers", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	unchanged := func() bool { return s == nil }
	if s != nil {
		originalAPI, originalClient := s.API, s.client
		unchanged = func() bool { return s.API == originalAPI && s.client == originalClient }
	}
	cfg := copyRouterUpdate(RouterUpdateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := cloudread.Context(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil cloud router update option"))
		}
		next := copyRouterUpdate(cfg)
		if err := errors.Join(apply(&next), cloudread.Context(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyRouterUpdate(next)
	}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		if !routerUpdateKey(key) {
			return nil, wrap(qosReadInvalid("%q is not a pinned cloud router update keyword", key))
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
	body, hasRoutes, err := routerUpdateBody(cfg)
	if err != nil {
		return nil, wrap(err)
	}
	if len(body) == 0 && !hasRoutes {
		return &RouterUpdateResult{NoOp: true}, nil
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
	result := &RouterUpdateResult{}
	if hasRoutes {
		private := &Service{client: &source.Client, API: selected}
		state, cause := qosReadStart(ctx, private, "extraroute")
		if state != nil {
			result.Extensions = append(result.Extensions, state.Extensions...)
		}
		if cause != nil {
			if !errors.Is(cause, ErrUnavailableExtension) {
				return result, wrap(errors.Join(cause, guard(ctx)))
			}
			result.RoutesDropped = true
		} else {
			body["routes"] = bytes.Clone(cfg.Attributes["routes"])
			ctx = rest.WithOperationGuard(ctx, state.Check)
		}
	}
	if len(body) == 0 {
		result.NoOp = true
		return result, wrap(guard(ctx))
	}

	result.Lookup, err = selected.Routers.FindRecord(ctx, identity)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	if result.Lookup == nil {
		return result, wrap(&resource.NotFoundError{Resource: "router", Reference: identity})
	}
	if result.Lookup.Resource == nil || result.Lookup.Wire == nil {
		return result, wrap(qosReadInvalid("router lookup did not produce a resource view"))
	}
	attrs := []routers.AttributeOption{}
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
		attrs = append(attrs, routers.WithAttribute(key, json.RawMessage(bytes.Clone(raw))))
	}
	if len(attrs) == 0 {
		result.NoOp = true
		return result, wrap(guard(ctx))
	}
	var id string
	if result.Lookup.Resource == nil || json.Unmarshal(result.Lookup.Resource.Body["id"], &id) != nil || id == "" {
		return result, wrap((&rest.Response{Body: result.Lookup.Envelope, Header: result.Lookup.Header, StatusCode: result.Lookup.StatusCode}).Fail(qosReadInvalid("router update requires a nonempty string response ID")))
	}
	if err := guard(ctx); err != nil {
		return result, wrap(err)
	}
	result.Router, err = selected.Routers.UpdateWithAttributes(ctx, id, attrs...)
	if err != nil {
		result.Router = nil // A rejected PUT is not a successful native result.
	}
	return result, wrap(errors.Join(err, guard(ctx)))
}
