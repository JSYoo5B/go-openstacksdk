package networks_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func firstPageNetworksListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*networks.Network, error) {
	return networks.New(client).Resources.All(ctx,
		resource.WithFilter("name", "server-condition"),
		resource.WithFilter("is_vlan_transparent", false),
		resource.WithFilter("created_at", "2025-01-02T03:04:05+00:00"),
		resource.WithPaginated(false),
	)
}
