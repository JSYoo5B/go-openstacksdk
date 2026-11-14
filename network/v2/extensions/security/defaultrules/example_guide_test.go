package defaultrules_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules"
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
	api := defaultrules.New(networkV2(ctx).RawClient())
	rule, err := api.Create(ctx,
		request.Field("direction", "ingress"),
		request.Field("ether_type", "IPv4"),
		request.Field("protocol", "tcp"),
		request.Field("port_range_min", 22),
		request.Field("port_range_max", 22),
		request.Field("used_in_default_sg", true))
	if err != nil {
		log.Fatal(err)
	}

	for item, err := range api.List(ctx, request.Query("ether_type", "IPv4")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.Text("id"), string(item.Resource.Body["port_range_min"]))
	}
	if _, err := api.Delete(ctx, rule.Text("id")); err != nil {
		log.Fatal(err)
	}
}
