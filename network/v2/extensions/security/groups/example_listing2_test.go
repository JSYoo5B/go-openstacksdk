package groups_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func firstPageSecurityGroupsListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*groups.SecGroup, error) {
	return groups.New(client).Resources.All(ctx,
		resource.WithFilter("name", "server-condition"),
		resource.WithFilter("stateful", false),
		resource.WithFilter("updated_at", "2025-01-03T03:04:05"),
		resource.WithPaginated(false),
	)
}
