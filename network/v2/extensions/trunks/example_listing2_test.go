package trunks_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func firstPageTrunksListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*trunks.Trunk, error) {
	return trunks.New(client).Resources.All(ctx,
		resource.WithFilter("name", "server-condition"),
		resource.WithFilter("status", "BUILD"),
		resource.WithFilter("tenant_id", nil),
		resource.WithFilter("fields", []string{"id", "name", "status", "tenant_id"}),
		resource.WithPaginated(false),
	)
}
