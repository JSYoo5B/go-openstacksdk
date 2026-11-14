package trunks_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listTrunksListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*trunks.Trunk, error) {
	return trunks.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"project_id":        "project-one",
			"is_admin_state_up": false,
			"fields":            []string{"id", "name", "project_id", "tenant_id"},
			"id":                "selected-trunk",
			"tenant_id":         "tenant-one",
		}),
		resource.WithMaxItems(20),
	)
}
