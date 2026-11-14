package subnetpools_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listTenantPoolsListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*subnetpools.SubnetPool, error) {
	return subnetpools.New(client).Resources.All(ctx,
		resource.WithFilter("tenant_id", "legacy-tenant"),
		resource.WithFilter("created_at", "2025-01-02T03:04:05+00:00"),
		resource.WithBodyFilter("minimum_prefix_length", 16),
		resource.WithPaginated(false),
	)
}
