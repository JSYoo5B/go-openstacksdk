package conntrackhelpers_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers"
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
	api := conntrackhelpers.New(networkV2(ctx).RawClient())
	routerID := "router-id"
	helper, err := api.Create(ctx, routerID, request.Field("helper", "ftp"), request.Field("protocol", "tcp"), request.Field("port", 21))
	if err != nil {
		log.Fatal(err)
	}
	id := helper.Text("id")
	if _, err := api.Update(ctx, routerID, id, request.Field("port", 2121)); err != nil {
		log.Fatal(err)
	}
	for item, err := range api.List(ctx, routerID, request.Query("helper", "ftp")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.Text("id"), string(item.Resource.Body["port"]))
	}
	if _, err := api.Delete(ctx, routerID, id); err != nil {
		log.Fatal(err)
	}
}
