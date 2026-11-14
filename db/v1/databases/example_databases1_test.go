package databases_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListDatabasesDatabases1(ctx context.Context, conn *sdk.Connection, instanceID string) error {
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Databases.InInstance(ctx, resource.ID(instanceID))
	if err != nil {
		return err
	}
	databases, err := scope.All(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false))
	if err != nil {
		return err
	}
	for _, database := range databases {
		fmt.Println(database.Name, database.CharSet, database.Collate)
	}
	return nil
}
