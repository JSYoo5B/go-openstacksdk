package flavors_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors"
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
	api := flavors.New(networkV2(ctx).RawClient())
	flavor, err := api.Create(ctx,
		request.Field("name", "gold"),
		request.Field("service_type", "L3_ROUTER_NAT"),
		request.Field("is_enabled", true))
	if err != nil {
		log.Fatal(err)
	}
	if _, err := api.AssociateServiceProfile(ctx, flavor.ID, "service-profile-id"); err != nil {
		log.Fatal(err)
	}
	for f, err := range api.List(ctx, request.Query("service_type", "L3_ROUTER_NAT")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(*f.Name, f.ServiceProfileIDs)
	}
	if _, err := api.Delete(ctx, flavor.ID); err != nil {
		log.Fatal(err)
	}
}
