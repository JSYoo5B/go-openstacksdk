package containers_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListContainersContainers1(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	for container, err := range service.Containers.Resources.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(container.Name, container.Count, container.Bytes)
	}
	return nil
}
