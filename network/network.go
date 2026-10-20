package network

import (
	"context"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/nativefind"
	"github.com/JSYoo5B/go-openstacksdk/internal/query"
	networkapi "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Network = networks.Network

type Service struct {
	API              *networkapi.Service
	Networks         *resource.Collection[Network]
	Ports            *resource.Collection[Port]
	FloatingIPs      *FloatingIPs
	Roles            *NetworkRoles
	client           *gophercloud.ServiceClient
	mutationNetworks *resource.Collection[Network]
	mutationRoles    *NetworkRoles
}

func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }

func New(client *gophercloud.ServiceClient) *Service {
	return NewWithDependencies(client, Dependencies{})
}

// Dependencies connects named server references to Compute. Connection
// supplies this resolver; explicit server IDs need no Compute lookup.
type Dependencies struct {
	Server       func(context.Context, resource.Ref) (string, error)
	NetworkRoles NetworkRolePolicy
}

// NewWithDependencies adds cross-service references while preserving New's
// standalone constructor. All collections share the supplied Neutron client.
func NewWithDependencies(client *gophercloud.ServiceClient, dependencies Dependencies) *Service {
	s := &Service{client: client, API: networkapi.New(client)}
	// The SDK supplies one complete list-filter policy for both facades. Keep
	// this convenience facade's singular errors, waiter and native query lane.
	adapter := s.API.Networks.ResourceAdapter()
	adapter.Kind = "network"
	adapter.Failed = func(status string) bool { return strings.EqualFold(status, "ERROR") }
	adapter.Get = func(ctx context.Context, id string) (*Network, error) { return networks.Get(ctx, client, id).Extract() }
	adapter.GetIdentityQuery = func(ctx context.Context, id string, q url.Values) (*Network, error) {
		var result networks.GetResult
		result.Header, result.Err = nativefind.Get(ctx, client, []string{"networks", id}, q, []int{200}, &result.Body)
		return result.Extract()
	}
	adapter.IterateControlled = nil
	adapter.List = func(q url.Values) pagination.Pager { return networks.List(client, query.Adapter(q)) }
	adapter.Extract = networks.ExtractNetworks
	adapter.Delete = func(ctx context.Context, id string) error {
		_, err := s.deleteNetworkID(ctx, id)
		return err
	}
	s.Networks = resource.NewCollection(adapter)
	s.Ports = s.API.Ports.Resources
	s.Roles = newNetworkRoles(client, dependencies.NetworkRoles)
	s.mutationNetworks, s.mutationRoles = s.Networks, s.Roles
	s.FloatingIPs = newFloatingIPs(s, dependencies)
	return s
}
