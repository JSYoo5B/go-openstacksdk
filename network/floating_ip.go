package network

import (
	"context"
	"fmt"
	"iter"
	"net/netip"
	"net/url"
	"sort"

	"gophercloudsdk/internal/query"
	floatingipapi "gophercloudsdk/network/v2/extensions/layer3/floatingips"
	portapi "gophercloudsdk/network/v2/ports"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	nativeports "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type FloatingIP = floatingipapi.FloatingIP
type Port = portapi.Port

// FloatingIPs combines common resource policies with Neutron allocation and
// server/port association. New allocations are never silently reused.
type FloatingIPs struct {
	*resource.Collection[FloatingIP]
	api              *floatingipapi.API
	ports            *portapi.API
	networks         *resource.Collection[Network]
	externalNetworks *resource.Collection[externalNetwork]
	roles            *NetworkRoles
	dependencies     Dependencies
	owner            *Service
}

type externalNetwork struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	External bool   `json:"router:external"`
}

func newFloatingIPs(s *Service, dependencies Dependencies) *FloatingIPs {
	external := resource.NewCollection(resource.Adapter[externalNetwork]{
		Kind:      "external network",
		ID:        func(n *externalNetwork) string { return n.ID },
		Name:      func(n *externalNetwork) string { return n.Name },
		NameQuery: func(name string) string { return name },
		Iterate: func(ctx context.Context, values url.Values) iter.Seq2[*externalNetwork, error] {
			values.Set("router:external", "true")
			pager := floatingIPSelectionPager(networks.List(s.client, query.Adapter(values)), func(r pagination.PageResult) pagination.Page {
				return networks.NetworkPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
			})
			return resource.Stream(ctx, pager, func(page pagination.Page) ([]externalNetwork, error) {
				var all []externalNetwork
				if err := networks.ExtractNetworksInto(page.(floatingIPSelectionPage).Page, &all); err != nil {
					return nil, err
				}
				filtered := make([]externalNetwork, 0, len(all))
				for _, network := range all {
					if network.External {
						filtered = append(filtered, network)
					}
				}
				return filtered, nil
			})
		},
	})
	return &FloatingIPs{
		Collection: s.API.FloatingIPs.Resources, api: s.API.FloatingIPs,
		ports: s.API.Ports, networks: s.Networks, externalNetworks: external,
		dependencies: dependencies, roles: s.Roles, owner: s,
	}
}

// Create allocates a new floating IP and optionally associates it with one
// destination IPv4 address. Automatic selection uses the shared NAT role when
// a server has multiple ports. Multiple eligible port/address pairs within the
// selected network return ErrAmbiguous before creation. Neutron
// errors do not trigger Nova fallback. A wait or association failure after
// allocation preserves the created resource alongside the error.
func (f *FloatingIPs) Create(ctx context.Context, input CreateFloatingIPRequest, options ...CreateFloatingIPOption) (*FloatingIP, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := input.Network.Validate(); err != nil {
		return nil, fmt.Errorf("external network: %w", err)
	}
	o := createFloatingIPOptions{}
	for _, apply := range options {
		if apply == nil {
			return nil, floatingIPInvalid("nil floating IP option")
		}
		if err := apply(&o); err != nil {
			return nil, err
		}
	}
	if o.server == nil && o.port == nil && (o.fixedAddress != "" || o.destination != nil || o.wait) {
		return nil, floatingIPInvalid("fixed address, NAT destination and ACTIVE waiting require WithServer or WithPort")
	}
	if err := o.validateFields(); err != nil {
		return nil, err
	}
	serverID := ""
	if o.server != nil {
		serverID = o.server.String()
		if o.server.IsName() {
			if f.dependencies.Server == nil {
				return nil, fmt.Errorf("%w: server resolver is unavailable", resource.ErrUnsupported)
			}
			id, err := f.dependencies.Server(ctx, *o.server)
			if err != nil {
				return nil, floatingIPWrap("resolve server", err)
			}
			serverID = id
		}
		if err := resource.ID(serverID).Validate(); err != nil {
			return nil, floatingIPWrap("resolve server", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	networkID, err := f.externalNetworks.ResolveID(ctx, input.Network)
	if err != nil {
		return nil, floatingIPWrap("resolve external network", err)
	}
	o.base.FloatingNetworkID = networkID
	if o.server != nil || o.port != nil {
		destination, err := f.selectDestination(ctx, serverID, o)
		if err != nil {
			return nil, floatingIPWrap("select destination", err)
		}
		o.base.PortID, o.base.FixedIP = destination.portID, destination.address
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	created, err := f.api.Create(ctx, o.base, o.fields...)
	if err != nil {
		return nil, floatingIPWrap("create", err)
	}
	if created == nil {
		return nil, floatingIPWrap("create", fmt.Errorf("Neutron returned no floating IP"))
	}
	if err := resource.ID(created.ID).Validate(); err != nil {
		return created, floatingIPWrap("verify creation", err)
	}
	if err := verifyFloatingIPDestination(created, o.base); err != nil {
		return created, floatingIPWrap("verify association", err)
	}
	if !o.wait {
		return created, nil
	}
	ready, err := f.Wait(ctx, resource.ID(created.ID), "ACTIVE", o.waitOptions...)
	if err != nil {
		return created, floatingIPWrap("create/wait", err)
	}
	if ready.ID != created.ID {
		return created, floatingIPWrap("verify creation", fmt.Errorf("wait returned floating IP %q; requested %q", ready.ID, created.ID))
	}
	if err := verifyFloatingIPDestination(ready, o.base); err != nil {
		return created, floatingIPWrap("verify association", err)
	}
	return ready, nil
}

type floatingIPDestination struct{ portID, address string }

func (f *FloatingIPs) selectDestination(ctx context.Context, serverID string, o createFloatingIPOptions) (floatingIPDestination, error) {
	networkID := ""
	if o.destination != nil {
		id, err := f.networks.ResolveID(ctx, *o.destination)
		if err != nil {
			return floatingIPDestination{}, err
		}
		networkID = id
	}
	var candidates []floatingIPDestination
	seen := map[floatingIPDestination]bool{}
	inspect := func(port *Port) error {
		if port == nil {
			return fmt.Errorf("Neutron returned an empty port")
		}
		if (serverID != "" && port.DeviceID != serverID) || (networkID != "" && port.NetworkID != networkID) {
			return nil
		}
		for _, fixed := range port.FixedIPs {
			address, err := netip.ParseAddr(fixed.IPAddress)
			if err != nil || !address.Is4() || (o.fixedAddress != "" && address.String() != o.fixedAddress) {
				continue
			}
			if err := resource.ID(port.ID).Validate(); err != nil {
				return err
			}
			candidate := floatingIPDestination{port.ID, address.String()}
			if !seen[candidate] {
				candidates = append(candidates, candidate)
				seen[candidate] = true
			}
		}
		return nil
	}
	if o.port != nil {
		port, err := f.ports.Resources.Find(ctx, *o.port)
		if err != nil {
			return floatingIPDestination{}, err
		}
		if !o.port.IsName() && (port == nil || port.ID != o.port.String()) {
			return floatingIPDestination{}, fmt.Errorf("Neutron response does not match requested port %q", o.port.String())
		}
		if err := inspect(port); err != nil {
			return floatingIPDestination{}, err
		}
	} else {
		filter := portapi.ListOpts{DeviceID: serverID, NetworkID: networkID}
		pager := floatingIPSelectionPager(nativeports.List(f.ports.RawClient(), filter), func(r pagination.PageResult) pagination.Page {
			return nativeports.PortPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
		})
		items := resource.Stream(ctx, pager, func(page pagination.Page) ([]Port, error) {
			return nativeports.ExtractPorts(page.(floatingIPSelectionPage).Page)
		})
		var serverPorts []*Port
		for port, err := range items {
			if err != nil {
				return floatingIPDestination{}, err
			}
			if port == nil {
				return floatingIPDestination{}, fmt.Errorf("Neutron returned an empty port")
			}
			if serverID != "" && port.DeviceID != serverID {
				continue
			}
			serverPorts = append(serverPorts, port)
		}
		// Count server-owned ports before filtering by IPv4. An IPv6-only
		// second port also requires a NAT role unless selection is explicit.
		if o.destination == nil && o.fixedAddress == "" && len(serverPorts) > 1 {
			roles, err := f.roles.Discover(ctx)
			if err != nil {
				return floatingIPDestination{}, err
			}
			if roles.NATDestination == nil {
				ids := make([]string, len(serverPorts))
				for i, port := range serverPorts {
					ids[i] = port.ID
				}
				sort.Strings(ids)
				return floatingIPDestination{}, &resource.AmbiguousError{Resource: "floating IP NAT destination", Name: serverID, IDs: ids}
			}
			networkID = roles.NATDestination.ID
			if err := resource.ID(networkID).Validate(); err != nil {
				return floatingIPDestination{}, err
			}
		}
		for _, port := range serverPorts {
			if err := inspect(port); err != nil {
				return floatingIPDestination{}, err
			}
		}
	}
	reference := serverID
	if o.port != nil {
		reference = o.port.String()
	}
	if len(candidates) == 0 {
		return floatingIPDestination{}, &resource.NotFoundError{Resource: "floating IP destination IPv4", Reference: reference}
	}
	if len(candidates) > 1 {
		ids := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			ids = append(ids, candidate.portID+"@"+candidate.address)
		}
		sort.Strings(ids)
		return floatingIPDestination{}, &resource.AmbiguousError{Resource: "floating IP destination", Name: reference, IDs: ids}
	}
	return candidates[0], nil
}

func verifyFloatingIPDestination(value *FloatingIP, expected floatingipapi.CreateOpts) error {
	if expected.PortID != "" && (value.PortID != expected.PortID || value.FixedIP != expected.FixedIP) {
		return fmt.Errorf("floating IP %q destination is %q/%q; requested %q/%q", value.ID, value.PortID, value.FixedIP, expected.PortID, expected.FixedIP)
	}
	return nil
}

func floatingIPWrap(operation string, err error) error {
	return &resource.OperationError{Operation: operation, Resource: "floating IP", Cause: err}
}
