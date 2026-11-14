package addressgroups_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listAddressGroupsListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*addressgroups.AddressGroup, error) {
	return addressgroups.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"project_id": "project-alpha",
			"name":       "server-name-filter",
			"addresses":  []string{"192.0.2.1", "198.51.100.0/24"},
		}),
		resource.WithMaxItems(20),
	)
}
