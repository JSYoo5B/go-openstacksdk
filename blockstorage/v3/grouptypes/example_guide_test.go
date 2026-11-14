package grouptypes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
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

func ExampleAPI_CreateGroupSpecs() {
	ctx := context.Background()
	api := grouptypes.New(blockStorageV3(ctx).RawClient())
	gt, err := api.Create(ctx, request.Field("name", "gold"), request.Field("is_public", true))
	if err != nil {
		log.Fatal(err)
	}
	specs, err := api.CreateGroupSpecs(ctx, gt.ID, map[string]any{"consistent_group_snapshot_enabled": "<is> True"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(specs)
	if err := api.Delete(ctx, gt.ID); err != nil {
		log.Fatal(err)
	}
}
