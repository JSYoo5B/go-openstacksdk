package labels_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func networkV2(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	api := labels.New(networkV2(ctx).RawClient())
	label, err := api.Create(ctx, request.Field("name", "web"), request.Field("is_shared", false))
	if err != nil {
		log.Fatal(err)
	}

	for item, err := range api.List(ctx, request.Query("project_id", "project-id")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(*item.Name, item.IsShared != nil && *item.IsShared)
	}
	if _, err := api.Delete(ctx, label.ID); err != nil {
		log.Fatal(err)
	}
}
