package attributes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/accelerator/v2"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func accelerator(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Accelerator(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	api := attributes.New(accelerator(ctx).RawClient())
	if _, err := api.Create(ctx, attributes.CreateOpts{DeployableID: 17, Key: "trait:CUSTOM_ACCEL", Value: "required"}); err != nil {
		log.Fatal(err)
	}
	for item, err := range api.List(ctx, resource.WithQuery("deployable_id", "17"), resource.WithQuery("key", "trait:CUSTOM_ACCEL")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.UUID, item.Value)
	}
}
