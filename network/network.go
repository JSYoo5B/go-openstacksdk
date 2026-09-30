package network

import (
	"context"
	networkapi "gophercloudsdk/network/v2"
	"net/url"
	"strings"

	"gophercloudsdk/internal/query"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Network = networks.Network

type Service struct {
	API      *networkapi.Service
	Networks *resource.Collection[Network]
	client   *gophercloud.ServiceClient
}

func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }

func New(client *gophercloud.ServiceClient) *Service {
	return &Service{client: client, API: networkapi.New(client), Networks: resource.NewCollection[Network](resource.Adapter[Network]{
		Kind:    "network",
		Get:     func(ctx context.Context, id string) (*Network, error) { return networks.Get(ctx, client, id).Extract() },
		List:    func(q url.Values) pagination.Pager { return networks.List(client, query.Adapter(q)) },
		Extract: networks.ExtractNetworks,
		Delete:  func(ctx context.Context, id string) error { return networks.Delete(ctx, client, id).ExtractErr() },
		ID:      func(n *Network) string { return n.ID }, Name: func(n *Network) string { return n.Name },
		NameQuery: func(name string) string { return name },
		Status:    func(n *Network) string { return n.Status },
		Failed:    func(status string) bool { return strings.EqualFold(status, "ERROR") },
	})}
}
