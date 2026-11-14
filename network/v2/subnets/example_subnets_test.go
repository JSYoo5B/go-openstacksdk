package subnets_test

import (
	"context"
	"fmt"

	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListSubnets(ctx context.Context, service *networkv2.Service) error {
	for subnet, err := range service.Subnets.Resources.List(ctx,
		resource.WithFilters(map[string]any{
			"network_id":      "network-a",
			"is_dhcp_enabled": false,
			"prefix_length":   24,
			"dns_nameservers": []string{"192.0.2.53"},
		}),
		resource.WithMaxItems(100),
	) {
		if err != nil {
			return err
		}
		fmt.Println(subnet.ID, subnet.CIDR)
	}
	return nil
}
