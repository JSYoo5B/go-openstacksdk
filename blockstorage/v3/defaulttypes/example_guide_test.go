package defaulttypes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes"
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

func ExampleAPI_Set() {
	ctx := context.Background()
	api := defaulttypes.New(blockStorageV3(ctx).RawClient())
	projectID, volumeTypeID := "project-id", "volume-type-id"
	if _, err := api.Set(ctx, projectID, volumeTypeID); err != nil {
		log.Fatal(err)
	}
	dt, err := api.Get(ctx, projectID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(*dt.VolumeTypeID)
	if err := api.Unset(ctx, projectID); err != nil {
		log.Fatal(err)
	}
}
