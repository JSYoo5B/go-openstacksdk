package policies_test

import (
	"context"
	"encoding/json"

	qospolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listTenantPoliciesListing2(ctx context.Context, client *gophercloud.ServiceClient, expectedRules json.RawMessage) ([]*qospolicies.Policy, error) {
	return qospolicies.New(client).Resources.All(ctx,
		resource.WithFilter("tenant_id", "legacy-tenant"),
		resource.WithBodyFilter("rules", expectedRules),
		resource.WithPaginated(false),
	)
}
