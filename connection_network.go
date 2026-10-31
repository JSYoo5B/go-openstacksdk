package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Invoked during Create, after the Compute constructor releases c.mu.
func (c *Connection) defaultServerNetwork(ctx context.Context) (resource.Ref, error) {
	if err := ctx.Err(); err != nil {
		return resource.Ref{}, err
	}
	if c.options.defaultNetworkSet {
		return c.options.defaultNetwork, nil
	}
	if c.options.configuredDefaultNetwork == "" {
		return resource.Ref{}, nil
	}
	roles, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return resource.Ref{}, err
	}
	// Disabled discovery and a missing catalog endpoint both yield no default.
	// Configured selectors and collisions are validated by shared discovery.
	if roles.DefaultNetwork == nil {
		return resource.Ref{}, ctx.Err()
	}
	ref := resource.ID(roles.DefaultNetwork.ID)
	if err := ref.Validate(); err != nil {
		return resource.Ref{}, err
	}
	return ref, ctx.Err()
}

// GetNetworkRoles shares one successful network/subnet discovery among all role
// getters. A missing catalog endpoint, or the network service disabled by
// ServiceEnabled (has_network: false), yields empty roles without a request,
// as Python's has_service('network') check does. HTTP/configuration
// failures remain errors and are not cached as an empty successful result.
func (c *Connection) GetNetworkRoles(ctx context.Context) (*network.NetworkRoleSnapshot, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// _find_interesting_networks: both discovery flags off, then has_service.
	if !c.UseExternalNetwork() && !c.UseInternalNetwork() {
		return &network.NetworkRoleSnapshot{}, nil
	}
	if !c.ServiceEnabled("network") {
		return &network.NetworkRoleSnapshot{}, nil
	}
	service, err := c.Network(ctx)
	if err != nil {
		if c.networkLookupUnavailable(ctx, err) {
			return &network.NetworkRoleSnapshot{}, ctx.Err()
		}
		return nil, err
	}
	return service.Roles.Discover(ctx)
}

// ResetNetworkRoles does not make an HTTP request or construct a service.
// It invalidates results for subsequent getters, without cancelling callers.
func (c *Connection) ResetNetworkRoles() {
	c.mu.Lock()
	service := c.network
	c.mu.Unlock()
	if service != nil {
		service.Roles.Reset()
	}
}

func (c *Connection) UseExternalNetwork() bool { return c.options.networkRoles.UseExternalNetwork() }

func (c *Connection) UseInternalNetwork() bool { return c.options.networkRoles.UseInternalNetwork() }

func (c *Connection) GetExternalIPv4Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.ExternalIPv4, nil
}

func (c *Connection) GetInternalIPv4Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.InternalIPv4, nil
}

func (c *Connection) GetExternalIPv6Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.ExternalIPv6, nil
}

func (c *Connection) GetInternalIPv6Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.InternalIPv6, nil
}

func (c *Connection) GetExternalIPv4FloatingNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.ExternalIPv4Floating, nil
}

func (c *Connection) GetExternalNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return append(s.ExternalIPv4, s.ExternalIPv6...), nil
}

func (c *Connection) GetInternalNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return append(s.InternalIPv4, s.InternalIPv6...), nil
}

func (c *Connection) GetNATSource(ctx context.Context) (*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.NATSource, nil
}

func (c *Connection) GetNATDestination(ctx context.Context) (*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.NATDestination, nil
}

func (c *Connection) GetDefaultNetwork(ctx context.Context) (*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.DefaultNetwork, nil
}

func (c *Connection) cloudNetworkListsContext(ctx context.Context, absentAllowed bool) (*network.Service, *resource.CloudLocation, error) {
	if err := c.recordListContext(ctx); err != nil {
		return nil, nil, err
	}
	service, err := c.Network(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, nil, errors.Join(err, cause)
		}
		if absentAllowed && missingCatalogEndpoint(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, nil, err
	}
	return service, &location, nil
}

// ListNetworkRecords returns [] for clean missing Neutron before filter options run.
func (c *Connection) ListNetworkRecords(ctx context.Context, options ...network.NetworkCloudListOption) (*network.NetworkCloudRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("ListNetworkRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudNetworkListsContext(ctx, true)
	if err != nil {
		return nil, wrap(err)
	}
	if service == nil {
		return &network.NetworkCloudRecords{Networks: make([]*networks.NetworkRecord, 0), Values: make([]any, 0), JSON: json.RawMessage("[]")}, nil
	}
	owned := append(slices.Clone(options), func(cfg *network.NetworkCloudListOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.ListNetworkRecords(ctx, owned...)
	return value, wrap(err)
}

func (c *Connection) SearchNetworkRecords(ctx context.Context, identity string, options ...network.NetworkCloudListOption) (*network.NetworkCloudRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchNetworkRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudNetworkListsContext(ctx, false)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.NetworkCloudListOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.SearchNetworkRecords(ctx, identity, owned...)
	return value, wrap(err)
}

func (c *Connection) ListSubnetRecords(ctx context.Context, options ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("ListSubnetRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudNetworkListsContext(ctx, true)
	if err != nil {
		return nil, wrap(err)
	}
	if service == nil {
		return &network.SubnetRecords{Subnets: make([]*subnets.SubnetRecord, 0), Inventory: make([]*subnets.SubnetRecord, 0), Value: make([]any, 0), JSON: json.RawMessage("[]")}, nil
	}
	owned := append(slices.Clone(options), func(cfg *network.SubnetReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.ListSubnetRecords(ctx, owned...)
	return value, wrap(err)
}

func (c *Connection) SearchSubnetRecords(ctx context.Context, identity string, options ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchSubnetRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudNetworkListsContext(ctx, false)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.SubnetReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.SearchSubnetRecords(ctx, identity, owned...)
	return value, wrap(err)
}

// ListPorts exposes the cached Neutron Source-shaped record list. A clean
// missing catalog endpoint returns a nonnil empty list before list options run,
// matching cloud.list_ports' has_service gate. Source polymorphic Proxy results
// remain available through NetworkV2.Ports.ProxyListRecords.
func (c *Connection) ListPorts(ctx context.Context, options ...ports.RecordListOption) ([]*ports.PortRecord, error) {
	wrap := func(err error) error { return request.Wrap("ListPorts", "ports", cloudread.ContextError(ctx, err)) }
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.NetworkV2(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if missingCatalogEndpoint(err) {
			return make([]*ports.PortRecord, 0), nil
		}
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	if _, err := location.ForResource(nil, nil); err != nil {
		return nil, wrap(err)
	}
	records, err := service.Ports.AllRecords(ctx, slices.Clone(options)...)
	if err != nil {
		return nil, wrap(err)
	}
	for _, record := range records {
		if err := cloudread.Context(ctx); err != nil {
			return nil, wrap(err)
		}
		actual, err := location.ForResource(record.Resource.Body["project_id"], record.Resource.Body["availability_zone"])
		if err != nil {
			return nil, wrap(err)
		}
		record.Resource.Body["location"] = actual
	}
	return records, wrap(cloudread.Context(ctx))
}

// SearchPorts injects recorded Connection facts before local name/mapping or
// expression selection. Actual foreign projects retain their IDs and clear the
// current project's names. A missing service still evaluates local expressions
// against [] without making a request or fabricating response evidence.
func (c *Connection) SearchPorts(ctx context.Context, nameOrID string, options ...ports.RecordSearchOption) (*ports.SearchResult, error) {
	wrap := func(err error) error { return request.Wrap("SearchPorts", "ports", cloudread.ContextError(ctx, err)) }
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.NetworkV2(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if missingCatalogEndpoint(err) {
			return c.searchAbsentPorts(ctx, nameOrID, slices.Clone(options))
		}
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), ports.WithRecordSearchCloudLocation(location))
	result, err := service.Ports.SearchRecords(ctx, nameOrID, owned...)
	return result, wrap(err)
}

func (c *Connection) searchAbsentPorts(ctx context.Context, nameOrID string, options []ports.RecordSearchOption) (*ports.SearchResult, error) {
	wrap := func(err error) error { return request.Wrap("SearchPorts", "ports", cloudread.ContextError(ctx, err)) }
	cfg := ports.RecordSearchOpts{}
	for _, apply := range options {
		if err := cloudread.Context(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(fmt.Errorf("%w: nil port search option", resource.ErrInvalidOption))
		}
		var next ports.RecordSearchOpts
		_ = ports.WithRecordSearchOpts(cfg)(&next)
		if err := errors.Join(apply(&next), cloudread.Context(ctx)); err != nil {
			return nil, wrap(err)
		}
		_ = ports.WithRecordSearchOpts(next)(&cfg)
	}
	result := &ports.SearchResult{Inventory: make([]*ports.PortRecord, 0)}
	selected, err := cloudfilter.SelectLoaded(make([]json.RawMessage, 0), nameOrID, cfg.Filters, func() error { return cloudread.Context(ctx) })
	if err != nil {
		return result, wrap(fmt.Errorf("%w: local port search: %w", resource.ErrInvalidOption, err))
	}
	result.Value = bytes.Clone(selected.Value)
	if !selected.Expression {
		result.Ports = make([]*ports.PortRecord, 0)
	}
	return result, wrap(cloudread.Context(ctx))
}

func (c *Connection) recordListContext(ctx context.Context) error {
	if err := cloudread.Context(ctx); err != nil {
		return err
	}
	if c == nil || c.provider == nil {
		return invalid("Connection is required")
	}
	return nil
}

// ListPortValues provides cloud.list_ports' complete list() conversion without
// changing ListPorts' typed compatibility API. A clean absent service returns
// [] before filters are expanded or callbacks run, as in the pinned Source.
func (c *Connection) ListPortValues(ctx context.Context, options ...ports.SourceCloudListOption) (*ports.SourceCloudListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListPortValues", "ports", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.NetworkV2(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if missingCatalogEndpoint(err) {
			return &ports.SourceCloudListResult{Values: make([]any, 0), JSON: json.RawMessage("[]"), Ports: make([]*ports.PortRecord, 0), Inventory: make([]*ports.PortRecord, 0)}, nil
		}
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), ports.WithSourceCloudListLocation(location))
	result, err := service.Ports.ListSourceValues(ctx, owned...)
	return result, wrap(err)
}

// CreateRouterRecord supplies cached Neutron and an owned default Connection location.
func (c *Connection) CreateRouterRecord(ctx context.Context, options ...network.RouterCreateOption) (*network.RouterCreateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("CreateRouterRecord", "network", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.Network(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.RouterCreateOpts) error {
		if cfg.CurrentLocation == nil {
			copy := location.Clone()
			cfg.CurrentLocation = &copy
		}
		return nil
	})
	value, err := service.CreateRouterRecord(ctx, owned...)
	return value, wrap(err)
}

// CreateSubnetRecord supplies cached Neutron and default location for owned lookup/create.
func (c *Connection) CreateSubnetRecord(ctx context.Context, networkName string, options ...network.SubnetCreateOption) (*network.SubnetCreateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("CreateSubnetRecord", "network", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.Network(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.SubnetCreateOpts) error {
		if cfg.CurrentLocation == nil {
			copy := location.Clone()
			cfg.CurrentLocation = &copy
		}
		return nil
	})
	value, err := service.CreateSubnetRecord(ctx, networkName, owned...)
	return value, wrap(err)
}

// ListRouterInterfaces supplies cached Neutron and immutable Connection location
// before the SDK classifies the complete observed port inventory.
func (c *Connection) ListRouterInterfaces(ctx context.Context, router json.RawMessage, options ...network.RouterInterfaceReadOption) (*network.RouterInterfaceReadResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListRouterInterfaces", "network", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.Network(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.RouterInterfaceReadOpts) error {
		if cfg.CurrentLocation == nil {
			copy := location.Clone()
			cfg.CurrentLocation = &copy
		}
		return nil
	})
	value, err := service.ListRouterInterfaces(ctx, router, owned...)
	return value, wrap(err)
}

// SecurityGroupCloud supplies SDK-owned default backend selection and immutable
// Connection location. The secgroup_source configuration (WithSecurityGroupSource
// or clouds.yaml, default "neutron") gates the feature like _has_secgroups: a
// source other than neutron or nova in any casing returns
// network.ErrUnavailableSecurityGroups before any request. Neutron is used only
// for the exact source "neutron" while the network service is enabled (has_network)
// and its endpoint exists; otherwise Nova is used. Only clean missing Neutron
// catalog endpoints select Nova; HTTP, cancellation, or joined discovery errors
// remain errors.
func (c *Connection) SecurityGroupCloud(ctx context.Context) (*network.SecurityGroupCloud, error) {
	wrap := func(err error) error {
		return request.Wrap("SecurityGroupCloud", "security groups", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	if _, err := location.ForResource(nil, nil); err != nil {
		return nil, wrap(err)
	}
	source := c.securityGroupSource()
	if !c.hasSecurityGroups() {
		return nil, wrap(network.ErrUnavailableSecurityGroups)
	}
	// _use_neutron_secgroups: has_service('network') and the exact source
	// "neutron". A disabled has_network flag selects Nova without a lookup.
	if source == "neutron" && c.ServiceEnabled("network") {
		service, err := c.Network(ctx)
		if err == nil {
			return network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Neutron: service, Source: &source, CurrentLocation: &location}), nil
		}
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if !missingCatalogEndpoint(err) {
			return nil, wrap(err)
		}
	}
	nova, err := c.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	absent := false
	return network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: nova.RawClient(), Source: &source, NetworkAvailable: &absent, CurrentLocation: &location}), nil
}

func (c *Connection) SearchSecurityGroupRecords(ctx context.Context, nameOrID string, options ...network.SecurityGroupReadOption) (*network.SecurityGroupReadResult, error) {
	cloud, err := c.SecurityGroupCloud(ctx)
	if err != nil {
		return nil, err
	}
	value, err := cloud.SearchSecurityGroups(ctx, nameOrID, slices.Clone(options)...)
	return value, request.Wrap("SearchSecurityGroupRecords", "security groups", cloudread.ContextError(ctx, err))
}

func (c *Connection) GetSecurityGroupRecord(ctx context.Context, reference any, options ...network.SecurityGroupReadOption) (*network.SecurityGroupReadResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetSecurityGroupRecord", "security groups", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	if value, handled, err := network.ExistingSecurityGroupRead(reference); handled || err != nil {
		return value, wrap(err)
	}
	cloud, err := c.SecurityGroupCloud(ctx)
	if err != nil {
		return nil, err
	}
	value, err := cloud.GetSecurityGroup(ctx, reference, slices.Clone(options)...)
	return value, request.Wrap("GetSecurityGroupRecord", "security groups", cloudread.ContextError(ctx, err))
}
