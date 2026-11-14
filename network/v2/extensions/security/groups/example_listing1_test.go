package groups_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listSecurityGroupsListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*groups.SecGroup, error) {
	return groups.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"is_shared":            false,
			"project_id":           "project-one",
			"fields":               []string{"id", "name", "created_at", "updated_at", "security_group_rules"},
			"created_at":           "2025-01-02T03:04:05+00:00",
			"security_group_rules": []any{},
		}),
		resource.WithMaxItems(20),
	)
}
