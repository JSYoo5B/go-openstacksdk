package actions_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterActionsActions1(ctx context.Context, conn *sdk.Connection, ownerID string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for action, err := range service.Actions.List(ctx,
		actions.WithListFilter("owner_id", ownerID),
		actions.WithListFilter("inputs", map[string]any{"region": "region-one"}),
		request.MaxItems(50), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(action.ID, action.Status)
	}
	return nil
}
