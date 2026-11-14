package subnetpools_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listPoolsListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*subnetpools.SubnetPool, error) {
	return subnetpools.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"is_shared":             false,
			"fields":                []string{"id", "name", "prefixes", "default_prefixlen", "min_prefixlen", "max_prefixlen"},
			"prefixes":              []string{"192.0.2.1/24"},
			"default_prefix_length": 24,
		}),
		resource.WithMaxItems(20),
	)
}
