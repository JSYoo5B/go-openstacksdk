package accounts_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
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
	api := accounts.New(objectStorage(ctx).RawClient())
	result, err := api.SetMetadataAndRefresh(ctx, map[string]any{
		"owner":                   "ops",
		"enabled":                 false,
		"X-Account-Meta-External": "value",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Refreshed.Metadata.Values["owner"])
}
