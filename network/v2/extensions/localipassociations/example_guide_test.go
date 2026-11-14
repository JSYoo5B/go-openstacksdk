package localipassociations_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations"
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
	api := localipassociations.New(networkV2(ctx).RawClient())
	lipID, portID := "local-ip-id", "port-id"
	if _, err := api.Create(ctx, lipID, request.Field("fixed_port_id", portID)); err != nil {
		log.Fatal(err)
	}

	for item, err := range api.List(ctx, lipID, request.Query("host", "compute-1")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(*item.FixedIP)
	}
	if err := api.Delete(ctx, lipID, portID); err != nil {
		log.Fatal(err)
	}
}
