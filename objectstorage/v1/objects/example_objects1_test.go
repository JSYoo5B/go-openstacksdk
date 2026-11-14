package objects_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListObjectsObjects1(ctx context.Context, conn *sdk.Connection, containerName string) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Objects.InContainer(ctx, resource.ID(containerName))
	if err != nil {
		return err
	}
	for object, err := range scope.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(object.Container, object.Name, object.Subdir, object.Bytes)
	}
	return nil
}
