package routers_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listRoutersListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*routers.Router, error) {
	return routers.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"project_id":       "project-one",
			"fields":           []string{"id", "name", "revision", "revision_number", "enable_ndp_proxy"},
			"enable_ndp_proxy": true,
			"revision_number":  7,
		}),
		resource.WithMaxItems(20),
	)
}
