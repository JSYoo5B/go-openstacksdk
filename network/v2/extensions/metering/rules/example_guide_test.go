package rules_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules"
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
	api := rules.New(networkV2(ctx).RawClient())
	labelID := "metering-label-id"
	rule, err := api.Create(ctx,
		request.Field("metering_label_id", labelID),
		request.Field("direction", "egress"),
		request.Field("destination_ip_prefix", "0.0.0.0/0"),
		request.Field("is_excluded", false))
	if err != nil {
		log.Fatal(err)
	}

	for item, err := range api.List(ctx, request.Query("metering_label_id", labelID)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(*item.Direction, *item.DestinationIPPrefix)
	}
	if _, err := api.Delete(ctx, rule.ID); err != nil {
		log.Fatal(err)
	}
}
