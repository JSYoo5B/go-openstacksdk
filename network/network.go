package network

import (
	"context"
	"encoding/json"
	"fmt"
	networkapi "gophercloudsdk/network/v2"
	"net/url"
	"strings"

	"gophercloudsdk/internal/nativefind"
	"gophercloudsdk/internal/query"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Network = networks.Network

type Service struct {
	API         *networkapi.Service
	Networks    *resource.Collection[Network]
	Ports       *resource.Collection[Port]
	FloatingIPs *FloatingIPs
	client      *gophercloud.ServiceClient
}

func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }

func New(client *gophercloud.ServiceClient) *Service {
	return NewWithDependencies(client, Dependencies{})
}

// Dependencies connects named server references to Compute. Connection
// supplies this resolver; explicit server IDs need no Compute lookup.
type Dependencies struct {
	Server func(context.Context, resource.Ref) (string, error)
}

// NewWithDependencies adds cross-service references while preserving New's
// standalone constructor. All collections share the supplied Neutron client.
func NewWithDependencies(client *gophercloud.ServiceClient, dependencies Dependencies) *Service {
	s := &Service{client: client, API: networkapi.New(client), Networks: resource.NewCollection[Network](resource.Adapter[Network]{
		Kind:             "network",
		IdentityFind:     true,
		BodyFilterFields: map[string]string{"subnets": "subnets", "subnet_ids": "subnets"},
		BodyFilterValue: func(n *Network, key string) (json.RawMessage, error) {
			if n == nil {
				return nil, fmt.Errorf("%w: nil body filter resource", resource.ErrInvalidOption)
			}
			if key == "subnets" {
				return json.Marshal(n.Subnets)
			}
			return nil, fmt.Errorf("%w: unsupported body filter field %q", resource.ErrInvalidOption, key)
		},
		Get: func(ctx context.Context, id string) (*Network, error) { return networks.Get(ctx, client, id).Extract() },
		GetIdentityQuery: func(ctx context.Context, id string, q url.Values) (*Network, error) {
			var result networks.GetResult
			result.Header, result.Err = nativefind.Get(ctx, client, []string{"networks", id}, q, []int{200}, &result.Body)
			return result.Extract()
		},
		List:    func(q url.Values) pagination.Pager { return networks.List(client, query.Adapter(q)) },
		Extract: networks.ExtractNetworks,
		Delete:  func(ctx context.Context, id string) error { return networks.Delete(ctx, client, id).ExtractErr() },
		ID:      func(n *Network) string { return n.ID }, Name: func(n *Network) string { return n.Name },
		NameQuery: func(name string) string { return name },
		Status:    func(n *Network) string { return n.Status },
		Failed:    func(status string) bool { return strings.EqualFold(status, "ERROR") },
	})}
	s.Ports = s.API.Ports.Resources
	s.FloatingIPs = newFloatingIPs(s, dependencies)
	return s
}
