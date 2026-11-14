package policies_test

import (
	"context"

	qospolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listPoliciesWithoutRulesListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*qospolicies.Policy, error) {
	return qospolicies.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"is_shared": false,
			"fields":    []string{"id", "name", "rules", "shared"},
			"rules":     []any{},
		}),
		resource.WithMaxItems(20),
	)
}
