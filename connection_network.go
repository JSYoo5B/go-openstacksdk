package openstack

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudnetwork"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// cloudNetworkDependencies fills the network workflow dependencies. The
// cached Network service, its lock and the role cache it owns stay on the
// Connection.
func (c *Connection) cloudNetworkDependencies() cloudnetwork.Dependencies {
	return cloudnetwork.Dependencies{
		Guard:                  c.recordListContext,
		Location:               c.CurrentLocation,
		Network:                c.Network,
		NetworkV2:              c.NetworkV2,
		Compute:                c.Compute,
		ServiceEnabled:         c.ServiceEnabled,
		UseExternalNetwork:     c.UseExternalNetwork,
		UseInternalNetwork:     c.UseInternalNetwork,
		MissingCatalogEndpoint: missingCatalogEndpoint,
		LookupUnavailable:      c.networkLookupUnavailable,
		SecurityGroupSource:    c.securityGroupSource,
		HasSecurityGroups:      c.hasSecurityGroups,
	}
}

// Invoked during Create, after the Compute constructor releases c.mu.
func (c *Connection) defaultServerNetwork(ctx context.Context) (resource.Ref, error) {
	return cloudnetwork.DefaultServerNetwork(ctx, c.options.defaultNetwork, c.options.defaultNetworkSet, c.options.configuredDefaultNetwork, c.GetNetworkRoles)
}

// GetNetworkRoles shares one successful network/subnet discovery among all role
// getters. A missing catalog endpoint, or the network service disabled by
// ServiceEnabled (has_network: false), yields empty roles without a request,
// as Python's has_service('network') check does. HTTP/configuration
// failures remain errors and are not cached as an empty successful result.
func (c *Connection) GetNetworkRoles(ctx context.Context) (*network.NetworkRoleSnapshot, error) {
	return cloudnetwork.GetNetworkRoles(ctx, c.cloudNetworkDependencies())
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
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) []*network.RoleNetwork { return s.ExternalIPv4 })
}

func (c *Connection) GetInternalIPv4Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) []*network.RoleNetwork { return s.InternalIPv4 })
}

func (c *Connection) GetExternalIPv6Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) []*network.RoleNetwork { return s.ExternalIPv6 })
}

func (c *Connection) GetInternalIPv6Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) []*network.RoleNetwork { return s.InternalIPv6 })
}

func (c *Connection) GetExternalIPv4FloatingNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) []*network.RoleNetwork { return s.ExternalIPv4Floating })
}

func (c *Connection) GetExternalNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) []*network.RoleNetwork {
		return append(s.ExternalIPv4, s.ExternalIPv6...)
	})
}

func (c *Connection) GetInternalNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) []*network.RoleNetwork {
		return append(s.InternalIPv4, s.InternalIPv6...)
	})
}

func (c *Connection) GetNATSource(ctx context.Context) (*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) *network.RoleNetwork { return s.NATSource })
}

func (c *Connection) GetNATDestination(ctx context.Context) (*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) *network.RoleNetwork { return s.NATDestination })
}

func (c *Connection) GetDefaultNetwork(ctx context.Context) (*network.RoleNetwork, error) {
	return cloudnetwork.Role(ctx, c.GetNetworkRoles, func(s *network.NetworkRoleSnapshot) *network.RoleNetwork { return s.DefaultNetwork })
}

// ListNetworkRecords returns [] for clean missing Neutron before filter options run.
func (c *Connection) ListNetworkRecords(ctx context.Context, options ...network.NetworkCloudListOption) (*network.NetworkCloudRecords, error) {
	return cloudnetwork.ListNetworkRecords(ctx, c.cloudNetworkDependencies(), options...)
}

func (c *Connection) SearchNetworkRecords(ctx context.Context, identity string, options ...network.NetworkCloudListOption) (*network.NetworkCloudRecords, error) {
	return cloudnetwork.SearchNetworkRecords(ctx, c.cloudNetworkDependencies(), identity, options...)
}

func (c *Connection) ListSubnetRecords(ctx context.Context, options ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	return cloudnetwork.ListSubnetRecords(ctx, c.cloudNetworkDependencies(), options...)
}

func (c *Connection) SearchSubnetRecords(ctx context.Context, identity string, options ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	return cloudnetwork.SearchSubnetRecords(ctx, c.cloudNetworkDependencies(), identity, options...)
}

// ListPorts exposes the cached Neutron Source-shaped record list. A clean
// missing catalog endpoint returns a nonnil empty list before list options run,
// matching cloud.list_ports' has_service gate. Source polymorphic Proxy results
// remain available through NetworkV2.Ports.ProxyListRecords.
func (c *Connection) ListPorts(ctx context.Context, options ...ports.RecordListOption) ([]*ports.PortRecord, error) {
	return cloudnetwork.ListPorts(ctx, c.cloudNetworkDependencies(), options...)
}

// SearchPorts injects recorded Connection facts before local name/mapping or
// expression selection. Actual foreign projects retain their IDs and clear the
// current project's names. A missing service still evaluates local expressions
// against [] without making a request or fabricating response evidence.
func (c *Connection) SearchPorts(ctx context.Context, nameOrID string, options ...ports.RecordSearchOption) (*ports.SearchResult, error) {
	return cloudnetwork.SearchPorts(ctx, c.cloudNetworkDependencies(), nameOrID, options...)
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
	return cloudnetwork.ListPortValues(ctx, c.cloudNetworkDependencies(), options...)
}

// CreateRouterRecord supplies cached Neutron and an owned default Connection location.
func (c *Connection) CreateRouterRecord(ctx context.Context, options ...network.RouterCreateOption) (*network.RouterCreateResult, error) {
	return cloudnetwork.CreateRouterRecord(ctx, c.cloudNetworkDependencies(), options...)
}

// CreateSubnetRecord supplies cached Neutron and default location for owned lookup/create.
func (c *Connection) CreateSubnetRecord(ctx context.Context, networkName string, options ...network.SubnetCreateOption) (*network.SubnetCreateResult, error) {
	return cloudnetwork.CreateSubnetRecord(ctx, c.cloudNetworkDependencies(), networkName, options...)
}

// ListRouterInterfaces supplies cached Neutron and immutable Connection location
// before the SDK classifies the complete observed port inventory.
func (c *Connection) ListRouterInterfaces(ctx context.Context, router json.RawMessage, options ...network.RouterInterfaceReadOption) (*network.RouterInterfaceReadResult, error) {
	return cloudnetwork.ListRouterInterfaces(ctx, c.cloudNetworkDependencies(), router, options...)
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
	return cloudnetwork.SecurityGroupCloud(ctx, c.cloudNetworkDependencies())
}

func (c *Connection) SearchSecurityGroupRecords(ctx context.Context, nameOrID string, options ...network.SecurityGroupReadOption) (*network.SecurityGroupReadResult, error) {
	return cloudnetwork.SearchSecurityGroupRecords(ctx, c.SecurityGroupCloud, nameOrID, options...)
}

func (c *Connection) GetSecurityGroupRecord(ctx context.Context, reference any, options ...network.SecurityGroupReadOption) (*network.SecurityGroupReadResult, error) {
	return cloudnetwork.GetSecurityGroupRecord(ctx, c.cloudNetworkDependencies(), c.SecurityGroupCloud, reference, options...)
}
