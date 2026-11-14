package networks_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listNetworksListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*networks.Network, error) {
	return networks.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"is_shared":  false,
			"fields":     []string{"id", "name", "subnets", "is_default", "mtu"},
			"subnet_ids": []string{"subnet-one"},
			"is_default": true,
			"mtu":        1500,
		}),
		resource.WithMaxItems(20),
	)
}
