package routers

import (
	"bytes"
	"encoding/json"
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RouterInterfaceRef preserves a supplied JSON reference and whether it came
// from a cached SDK resource. A resource is truthy even when its ID is null;
// ordinary JSON references use Source Python truthiness before _get_id.
type RouterInterfaceRef struct {
	Value    json.RawMessage
	Resource bool
}

// RouterInterfaceOpts follows the named Source proxy arguments. Port wins
// Subnet when truthy. The deprecated PortID/SubnetID aliases override their
// respective current names regardless of option order. AdvertiseHost is add-
// only and is emitted only when truthy. Headers are an explicit Go convenience.
type RouterInterfaceOpts struct {
	Port, Subnet, PortID, SubnetID *RouterInterfaceRef
	AdvertiseHost                  json.RawMessage
	Headers                        map[string]string
}

// RouterInterfaceOption is one option of AddInterfaceRecord,
// RemoveInterfaceRecord and RemoveCloudInterfaceRecord: WithRouterInterfaceOpts,
// WithRouterInterfacePort, WithRouterInterfaceSubnet,
// WithRouterInterfacePortID, WithRouterInterfaceSubnetID,
// WithRouterInterfaceAdvertiseHost, request.Headers (replaces the headers) or
// request.Custom over *RouterInterfaceOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RouterInterfaceOption = request.WriteCall

func copyRouterInterfaceRef(value *RouterInterfaceRef) *RouterInterfaceRef {
	if value == nil {
		return nil
	}
	return &RouterInterfaceRef{Value: bytes.Clone(value.Value), Resource: value.Resource}
}
func copyRouterInterfaceOpts(value RouterInterfaceOpts) RouterInterfaceOpts {
	value.Port, value.Subnet = copyRouterInterfaceRef(value.Port), copyRouterInterfaceRef(value.Subnet)
	value.PortID, value.SubnetID = copyRouterInterfaceRef(value.PortID), copyRouterInterfaceRef(value.SubnetID)
	value.AdvertiseHost, value.Headers = bytes.Clone(value.AdvertiseHost), maps.Clone(value.Headers)
	return value
}
func routerInterfaceRef(value any) (*RouterInterfaceRef, error) {
	var cached *resource.RawResource
	switch value := value.(type) {
	case *resource.RawResource:
		cached = value
	case resource.RawResource:
		cached = &value
	default:
		raw, err := json.Marshal(value)
		return &RouterInterfaceRef{Value: bytes.Clone(raw)}, err
	}
	if cached == nil {
		return &RouterInterfaceRef{Value: json.RawMessage("null")}, nil
	}
	raw := bytes.Clone(cached.Body["id"])
	if raw == nil {
		raw = json.RawMessage("null")
	}
	return &RouterInterfaceRef{Value: raw, Resource: true}, nil
}
func WithRouterInterfaceOpts(value RouterInterfaceOpts) RouterInterfaceOption {
	owned := copyRouterInterfaceOpts(value)
	return request.Custom(func(config *RouterInterfaceOpts) error { *config = copyRouterInterfaceOpts(owned); return nil })
}
func routerInterfaceReferenceOption(key string, value any) RouterInterfaceOption {
	owned, err := routerInterfaceRef(value)
	return request.Custom(func(config *RouterInterfaceOpts) error {
		if err != nil {
			return err
		}
		switch key {
		case "port":
			config.Port = copyRouterInterfaceRef(owned)
		case "subnet":
			config.Subnet = copyRouterInterfaceRef(owned)
		case "port_id":
			config.PortID = copyRouterInterfaceRef(owned)
		case "subnet_id":
			config.SubnetID = copyRouterInterfaceRef(owned)
		}
		return nil
	})
}
func WithRouterInterfacePort(value any) RouterInterfaceOption {
	return routerInterfaceReferenceOption("port", value)
}
func WithRouterInterfaceSubnet(value any) RouterInterfaceOption {
	return routerInterfaceReferenceOption("subnet", value)
}

// WithRouterInterfacePortID follows Source's deprecated port_id alias. When
// both names are supplied, this alias overrides port independently of order.
func WithRouterInterfacePortID(value any) RouterInterfaceOption {
	return routerInterfaceReferenceOption("port_id", value)
}
func WithRouterInterfaceSubnetID(value any) RouterInterfaceOption {
	return routerInterfaceReferenceOption("subnet_id", value)
}
func WithRouterInterfaceAdvertiseHost(value any) RouterInterfaceOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(config *RouterInterfaceOpts) error {
		if err != nil {
			return err
		}
		config.AdvertiseHost = bytes.Clone(owned)
		return nil
	})
}
func withRouterInterfaceHeaders(values map[string]string) func(*RouterInterfaceOpts) error {
	owned := maps.Clone(values)
	return func(config *RouterInterfaceOpts) error { config.Headers = maps.Clone(owned); return nil }
}

// RouterGatewayOpts provides complete single-gateway attributes. The default
// route bridges the pinned Source intent to stock Neutron external_gateway_info.
// LegacySourceRoutes explicitly selects the old Source action path; both modes
// use strict HTTP errors. A removal bridge accepts no ignored gateway fields.
type RouterGatewayOpts struct {
	Fields             map[string]json.RawMessage
	Headers            map[string]string
	LegacySourceRoutes bool
}

// RouterGatewayOption is one option of AddGatewayRecord and
// RemoveGatewayRecord: WithRouterGatewayOpts, WithSourceLegacyGatewayRoutes,
// request.Field (one gateway attribute; its marshal error is returned when
// applied), request.Headers (replaces the headers) or request.Custom over
// *RouterGatewayOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RouterGatewayOption = request.WriteCall

func copyRouterGatewayOpts(value RouterGatewayOpts) RouterGatewayOpts {
	fields := make(map[string]json.RawMessage, len(value.Fields))
	for key, raw := range value.Fields {
		fields[key] = bytes.Clone(raw)
	}
	value.Fields, value.Headers = fields, maps.Clone(value.Headers)
	return value
}
func WithRouterGatewayOpts(value RouterGatewayOpts) RouterGatewayOption {
	owned := copyRouterGatewayOpts(value)
	return request.Custom(func(config *RouterGatewayOpts) error { *config = copyRouterGatewayOpts(owned); return nil })
}

func withRouterGatewayField(key string, raw json.RawMessage, err error) func(*RouterGatewayOpts) error {
	owned := bytes.Clone(raw)
	return func(config *RouterGatewayOpts) error {
		if err != nil {
			return err
		}
		if config.Fields == nil {
			config.Fields = make(map[string]json.RawMessage)
		}
		config.Fields[key] = bytes.Clone(owned)
		return nil
	}
}
func withRouterGatewayHeaders(values map[string]string) func(*RouterGatewayOpts) error {
	owned := maps.Clone(values)
	return func(config *RouterGatewayOpts) error { config.Headers = maps.Clone(owned); return nil }
}

// WithSourceLegacyGatewayRoutes chooses pinned add_gateway_router or
// remove_gateway_router paths only for deployments which implement them. It
// does not turn HTTP failures into JSON success values as the Source bug does.
func WithSourceLegacyGatewayRoutes(value bool) RouterGatewayOption {
	return request.Custom(func(config *RouterGatewayOpts) error { config.LegacySourceRoutes = value; return nil })
}

// routerInterfaceCalls interprets the shared calls of RouterInterfaceOption.
var routerInterfaceCalls = calls.Spec[RouterInterfaceOpts]{
	Headers: neutronfind.Use(withRouterInterfaceHeaders),
}

// routerGatewayCalls interprets the shared calls of RouterGatewayOption.
var routerGatewayCalls = calls.Spec[RouterGatewayOpts]{
	Field:   neutronfind.UseJSON(withRouterGatewayField),
	Headers: neutronfind.Use(withRouterGatewayHeaders),
}
