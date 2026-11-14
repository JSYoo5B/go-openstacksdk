package networksegmentranges_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges"
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
	api := networksegmentranges.New(networkV2(ctx).RawClient())
	rng, err := api.Create(ctx,
		request.Field("name", "vlan-100"),
		request.Field("network_type", "vlan"),
		request.Field("physical_network", "physnet1"),
		request.Field("minimum", 100),
		request.Field("maximum", 199),
		request.Field("shared", true))
	if err != nil {
		log.Fatal(err)
	}
	id := rng.Text("id")
	if _, err := api.Update(ctx, id, request.Field("maximum", 299)); err != nil {
		log.Fatal(err)
	}
	for item, err := range api.List(ctx, request.Query("network_type", "vlan")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.Text("name"), string(item.Resource.Body["available"]))
	}
	if _, err := api.Delete(ctx, id); err != nil {
		log.Fatal(err)
	}
}
