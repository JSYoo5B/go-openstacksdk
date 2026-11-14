package policytypes_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterPolicyTypesPolicytypes1(ctx context.Context, conn *sdk.Connection, typeName string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	values, err := service.PolicyTypes.All(ctx,
		policytypes.WithListFilter("name", typeName),
		request.MaxItems(50), request.Paginated(false))
	if err != nil {
		return err
	}
	for _, policy := range values {
		fmt.Println(policy.Name, policy.Version)
	}
	return nil
}
