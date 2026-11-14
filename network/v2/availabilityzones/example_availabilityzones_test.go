package availabilityzones_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones"
)

func Example_availabilityzones() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := availabilityzones.New(service.RawClient())
	zones, err := api.AllRecords(ctx, availabilityzones.WithProxyQuery("name", "az-a"))
	if err != nil {
		log.Fatal(err)
	}
	for _, zone := range zones {
		fmt.Println(string(zone.Resource.Body["name"]), string(zone.Resource.Body["resource"]), string(zone.Resource.Body["state"]))
	}
}
