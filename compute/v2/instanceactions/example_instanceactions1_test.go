package instanceactions_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListServerActionsInstanceactions1(ctx context.Context, conn *sdk.Connection, serverID string) error {
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	scope, err := service.InstanceActions.InServer(ctx, resource.ID(serverID))
	if err != nil {
		return err
	}
	for action, err := range scope.List(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(action.ServerID, action.RequestID, action.Action)
	}
	return nil
}
