package clusterpolicies_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func FilterClusterPoliciesClusterpolicies1(ctx context.Context, conn *sdk.Connection, cluster resource.Ref) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.ClusterPolicies.InCluster(ctx, cluster)
	if err != nil {
		return err
	}
	for binding, err := range scope.List(ctx,
		clusterpolicies.WithListFilter("is_enabled", false),
		clusterpolicies.WithListFilter("data", map[string]any{"team": "infra"}),
		request.MaxItems(50), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(binding.ID, binding.PolicyID, binding.URIClusterID)
	}
	return nil
}
