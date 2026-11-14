package addressgroups_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listTenantAddressGroupsListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*addressgroups.AddressGroup, error) {
	return addressgroups.New(client).Resources.All(ctx,
		resource.WithFilter("tenant_id", "legacy-tenant"),
		resource.WithBodyFilter("addresses", []string{}),
		resource.WithPaginated(false),
	)
}
