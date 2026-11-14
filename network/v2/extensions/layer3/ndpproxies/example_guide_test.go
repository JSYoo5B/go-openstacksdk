package ndpproxies_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies"
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
	api := ndpproxies.New(networkV2(ctx).RawClient())
	routerID := "router-id"
	proxy, err := api.Create(ctx, request.Field("router_id", routerID), request.Field("port_id", "port-id"), request.Field("ip_address", "2001:db8::10"))
	if err != nil {
		log.Fatal(err)
	}
	id := proxy.Text("id")
	if _, err := api.Update(ctx, id, request.Field("description", "web")); err != nil {
		log.Fatal(err)
	}
	for item, err := range api.List(ctx, request.Query("router_id", routerID)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.Text("ip_address"))
	}
	if _, err := api.Delete(ctx, id); err != nil {
		log.Fatal(err)
	}
}
