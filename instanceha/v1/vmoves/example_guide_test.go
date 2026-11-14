package vmoves_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/instanceha/v1"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/vmoves"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func instanceHA(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.InstanceHAV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	client := instanceHA(ctx).RawClient()
	client.Microversion = "1.3"
	scope, err := vmoves.New(client).InNotification(ctx, resource.ID("notification-uuid"))
	if err != nil {
		log.Fatal(err)
	}
	state, kind := "ongoing", "evacuation"
	moves, err := scope.All(ctx, vmoves.WithListOptions(vmoves.ListOpts{Status: &state, Type: &kind}))
	if err != nil {
		log.Fatal(err)
	}
	for _, move := range moves {
		move, err = scope.Get(ctx, move.UUID)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(move)
	}
}
