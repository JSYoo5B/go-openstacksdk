package portpairgroups_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups"
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
	api := portpairgroups.New(networkV2(ctx).RawClient())
	ppg, err := api.Create(ctx,
		request.Field("name", "fw-group"),
		request.Field("port_pairs", []string{"port-pair-id"}),
		request.Field("port_pair_group_parameters", map[string]any{"lb_fields": []string{"ip_src"}}))
	if err != nil {
		log.Fatal(err)
	}

	if _, err := api.Update(ctx, ppg.ID, request.Field("port_pairs", []string{"port-pair-id", "port-pair-2-id"})); err != nil {
		log.Fatal(err)
	}
	fmt.Println(ppg.ID)
	if err := api.Delete(ctx, ppg.ID); err != nil {
		log.Fatal(err)
	}
}
