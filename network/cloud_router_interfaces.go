package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RouterInterfaceReadOpts owns the pinned interface discriminator. Omission and
// any falsey JSON value select both internal and external interfaces.
type RouterInterfaceReadOpts struct {
	InterfaceType   *json.RawMessage
	Headers         map[string]string
	Microversion    *string
	CurrentLocation *resource.CloudLocation
}
type RouterInterfaceReadOption func(*RouterInterfaceReadOpts) error

func copyRouterInterfaceRead(v RouterInterfaceReadOpts) RouterInterfaceReadOpts {
	if v.InterfaceType != nil {
		x := json.RawMessage(bytes.Clone(*v.InterfaceType))
		v.InterfaceType = &x
	}
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = map[string]string{}
	}
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
func WithRouterInterfaceReadOpts(v RouterInterfaceReadOpts) RouterInterfaceReadOption {
	owned := copyRouterInterfaceRead(v)
	return func(c *RouterInterfaceReadOpts) error { *c = copyRouterInterfaceRead(owned); return nil }
}
func WithRouterInterfaceTypeJSON(v json.RawMessage) RouterInterfaceReadOption {
	owned := bytes.Clone(v)
	return func(c *RouterInterfaceReadOpts) error {
		x := json.RawMessage(bytes.Clone(owned))
		c.InterfaceType = &x
		return nil
	}
}
func WithRouterInterfaceType(v string) RouterInterfaceReadOption {
	raw, _ := json.Marshal(v)
	return WithRouterInterfaceTypeJSON(raw)
}
func WithRouterInterfaceReadHeaders(v map[string]string) RouterInterfaceReadOption {
	owned := maps.Clone(v)
	return func(c *RouterInterfaceReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRouterInterfaceReadMicroversion(v string) RouterInterfaceReadOption {
	return func(c *RouterInterfaceReadOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRouterInterfaceReadCurrentLocation(v resource.CloudLocation) RouterInterfaceReadOption {
	owned := v.Clone()
	return func(c *RouterInterfaceReadOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}

// RouterInterfaceRouterID constructs the Source router dictionary without a
// lookup or a caller-defined builder. Raw dictionary inputs retain null/numeric IDs.
func RouterInterfaceRouterID(id string) json.RawMessage {
	raw, _ := json.Marshal(map[string]string{"id": id})
	return raw
}

type RouterInterfaceReadResult struct {
	Ports     []*ports.PortRecord
	Inventory []*ports.PortRecord
	LastPort  *ports.PortRecord
	Value     json.RawMessage
}

func routerInterfaceRecord(v *ports.PortRecord) *ports.PortRecord {
	if v == nil {
		return nil
	}
	c := *v
	c.Resource = v.Resource.Clone()
	c.Wire = v.Wire.Clone()
	c.Envelope = bytes.Clone(v.Envelope)
	c.Header = v.Header.Clone()
	return &c
}
func routerInterfaceProof(v *ports.PortRecord, e error) error {
	if v == nil {
		return e
	}
	return (&rest.Response{Body: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}).Fail(e)
}

// cloudNetworkReadLocation applies Source loaded truthiness/equality without
// changing the resource's untyped project ID or the legacy location policy.
func cloudNetworkReadLocation(body map[string]json.RawMessage, location *resource.CloudLocation) (json.RawMessage, error) {
	if location == nil {
		return json.RawMessage("null"), nil
	}
	project := body["project_id"]
	if project != nil {
		truthy, e := cloudfilter.PythonJSONTruthy(project)
		if e != nil {
			return nil, e
		}
		if !truthy {
			project = nil
		} else {
			current := location.Project.ID
			if current == nil {
				current = json.RawMessage("null")
			}
			equal, e := jsonfilter.EqualPythonLoadedJSON(project, current)
			if e != nil {
				return nil, e
			}
			if equal {
				project = nil
			}
		}
	}
	// Ports and security groups declare project_id but not availability_zone.
	return location.ForResource(project, nil)
}

// ListRouterInterfaces reads router["id"] literally, collects every device_id
// port page, then returns internal rows before gateway rows in Source order.
// It never resolves a router name or issues a router GET.
func (s *Service) ListRouterInterfaces(ctx context.Context, router json.RawMessage, options ...RouterInterfaceReadOption) (*RouterInterfaceReadResult, error) {
	const op = "ListRouterInterfaces"
	wrap := func(e error) error { return request.Wrap(op, "router interfaces", cloudread.ContextError(ctx, e)) }
	if e := cloudread.Context(ctx); e != nil {
		return nil, wrap(e)
	}
	if s == nil || s.API == nil || s.client == nil || s.API.RawClient() != s.client {
		return nil, wrap(fmt.Errorf("%w: configured network service required", resource.ErrInvalidOption))
	}
	originalAPI, originalClient := s.API, s.client
	source, e := cloudread.Capture(ctx, originalClient, "network")
	if e != nil {
		return nil, wrap(e)
	}
	guard := func(ctx context.Context) error {
		if s.API != originalAPI || s.client != originalClient || originalAPI.RawClient() != s.client {
			return fmt.Errorf("%w: selected network service/API changed", resource.ErrInvalidOption)
		}
		return source.Guard(ctx)
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	cfg := copyRouterInterfaceRead(RouterInterfaceReadOpts{})
	for _, apply := range slices.Clone(options) {
		if e := errors.Join(guard(ctx), cloudread.Context(ctx)); e != nil {
			return nil, wrap(e)
		}
		if apply == nil {
			return nil, wrap(fmt.Errorf("%w: nil router interface option", resource.ErrInvalidOption))
		}
		next := copyRouterInterfaceRead(cfg)
		if e := errors.Join(apply(&next), guard(ctx), cloudread.Context(ctx)); e != nil {
			return nil, wrap(e)
		}
		cfg = copyRouterInterfaceRead(next)
	}
	ownedRouter := bytes.Clone(router)
	members, e := cloudfilter.ObjectMembers(ownedRouter)
	if e != nil {
		return nil, wrap(e)
	}
	var id json.RawMessage
	found := false
	for _, member := range members {
		if member.Key == "id" {
			id = bytes.Clone(member.Value)
			found = true
		}
	}
	if !found {
		return nil, wrap(fmt.Errorf("%w: router dictionary has no id", resource.ErrInvalidOption))
	}
	api := networkv2.New(&source.Client)
	leaf := []ports.RecordListOption{ports.WithRecordListQuery("device_id", id), ports.WithRecordListHeaders(cfg.Headers)}
	if cfg.Microversion != nil {
		leaf = append(leaf, ports.WithRecordListMicroversion(*cfg.Microversion))
	}
	result := &RouterInterfaceReadResult{Inventory: make([]*ports.PortRecord, 0)}
	fail := func(e error) (*RouterInterfaceReadResult, error) {
		result.Value = nil
		result.Ports = nil
		return result, wrap(e)
	}
	for record, e := range api.Ports.ListRecords(ctx, leaf...) {
		if record != nil {
			result.LastPort = routerInterfaceRecord(record)
		}
		if e = errors.Join(e, guard(ctx), cloudread.Context(ctx)); e != nil {
			return fail(e)
		}
		if record == nil || record.Resource == nil {
			return fail(fmt.Errorf("%w: missing port Resource", resource.ErrInvalidOption))
		}
		raw, e := cloudNetworkReadLocation(record.Resource.Body, cfg.CurrentLocation)
		if e != nil {
			return fail(routerInterfaceProof(record, e))
		}
		record.Resource.Body["location"] = raw
		result.Inventory = append(result.Inventory, routerInterfaceRecord(record))
	}
	internal, external := true, true
	if cfg.InterfaceType != nil {
		truthy, e := cloudfilter.PythonJSONTruthy(*cfg.InterfaceType)
		if e != nil {
			return fail(routerInterfaceProof(result.LastPort, e))
		}
		if truthy {
			internal = bytes.Equal(bytes.TrimSpace(*cfg.InterfaceType), []byte(`"internal"`))
			external = bytes.Equal(bytes.TrimSpace(*cfg.InterfaceType), []byte(`"external"`))
			// JSON escaped spellings are equivalent Python strings.
			var text string
			if json.Unmarshal(*cfg.InterfaceType, &text) == nil {
				internal = text == "internal"
				external = text == "external"
			}
		}
	}
	result.Ports = make([]*ports.PortRecord, 0)
	for _, wantInternal := range []bool{true, false} {
		if wantInternal && !internal || !wantInternal && !external {
			continue
		}
		for _, record := range result.Inventory {
			if e := errors.Join(guard(ctx), cloudread.Context(ctx)); e != nil {
				return fail(e)
			}
			var owner string
			_ = json.Unmarshal(record.Resource.Body["device_owner"], &owner)
			include := owner == "network:router_gateway"
			if wantInternal {
				include = owner == "network:router_interface" || owner == "network:router_interface_distributed" || owner == "network:ha_router_replicated_interface"
			}
			if include {
				result.Ports = append(result.Ports, routerInterfaceRecord(record))
			}
		}
	}
	values := make([]map[string]json.RawMessage, len(result.Ports))
	for i, record := range result.Ports {
		values[i] = record.Resource.Body
	}
	result.Value, e = json.Marshal(values)
	if e = errors.Join(e, guard(ctx), cloudread.Context(ctx)); e != nil {
		return fail(e)
	}
	return result, nil
}
