package routers_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func firstPageRoutersListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*routers.Router, error) {
	return routers.New(client).Resources.All(ctx,
		resource.WithFilter("name", "server-condition"),
		resource.WithFilter("external_gateway_info", map[string]any{
			"network_id": "external-network",
		}),
		resource.WithFilter("created_at", "2025-01-02T03:04:05+00:00"),
		resource.WithPaginated(false),
	)
}
