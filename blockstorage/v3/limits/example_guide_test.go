package limits_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/limits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func blockStorageV3(ctx context.Context) *v3.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_InProject() {
	ctx := context.Background()
	api := limits.New(blockStorageV3(ctx).RawClient())
	current, err := api.Fetch(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(current.ProjectID)
	scope, err := api.InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	result, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Absolute.MaxTotalVolumes)
}
