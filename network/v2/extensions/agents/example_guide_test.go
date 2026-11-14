package agents_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
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

func Example_listDHCPNetworks() {
	ctx := context.Background()
	api := agents.New(networkV2(ctx).RawClient())
	for row, err := range api.ListDHCPNetworkRecords(ctx, "agent-id",
		request.Query("is_shared", true),
		request.Query("fields", []string{"id", "name"}),
		request.Query("mtu", 1450),
	) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(row)
	}
}
