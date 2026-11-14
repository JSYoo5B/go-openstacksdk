package users_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListUsersUsers1(ctx context.Context, conn *sdk.Connection, instanceID, host string) error {
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Users.InInstance(ctx,
		resource.ID(instanceID), users.WithHost(host))
	if err != nil {
		return err
	}
	for user, err := range scope.List(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(user.Name, user.Host, user.Databases)
	}
	return nil
}
