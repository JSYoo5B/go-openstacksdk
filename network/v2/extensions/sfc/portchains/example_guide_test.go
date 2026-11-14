package portchains_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains"
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
	api := portchains.New(networkV2(ctx).RawClient())
	pc, err := api.Create(ctx,
		request.Field("name", "web-chain"),
		request.Field("port_pair_groups", []string{"port-pair-group-id"}),
		request.Field("flow_classifiers", []string{"flow-classifier-id"}),
		request.Field("chain_parameters", map[string]any{"correlation": "mpls", "symmetric": false}))
	if err != nil {
		log.Fatal(err)
	}

	if _, err := api.Update(ctx, pc.ID, request.Field("flow_classifiers", []string{"flow-classifier-id", "flow-classifier-2-id"})); err != nil {
		log.Fatal(err)
	}
	fmt.Println(pc.ID)
	if err := api.Delete(ctx, pc.ID); err != nil {
		log.Fatal(err)
	}
}
