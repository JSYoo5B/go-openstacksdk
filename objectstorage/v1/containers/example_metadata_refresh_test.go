package containers_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func objectStorage(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ObjectStorageV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_SetMetadataAndRefresh() {
	ctx := context.Background()
	api := containers.New(objectStorage(ctx).RawClient())
	result, err := api.SetMetadataAndRefresh(ctx, "container", map[string]any{
		"owner":                     "ops",
		"enabled":                   false,
		"X-Container-Meta-External": "value",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Refreshed.Metadata.Values["owner"])
}
