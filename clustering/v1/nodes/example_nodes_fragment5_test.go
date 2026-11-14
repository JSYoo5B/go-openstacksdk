package nodes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_check() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ClusteringV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	api := nodes.New(client)
	checked, err := api.Check(ctx, resource.ID("NODE_ID"))
	if err != nil {
		log.Fatal(err)
	}
	recovered, err := api.Recover(ctx, resource.ID("NODE_ID"), nodes.RecoverOpts{},
		request.Custom(func(c *request.Config[nodes.RecoverOpts]) error {
			c.Options.Operation = request.Present("REBOOT")
			return nil
		}), nodes.WithRecoverOperationParams(map[string]any{}),
		request.Custom(func(c *request.Config[nodes.RecoverOpts]) error { c.Options.Check = request.Present(false); return nil }))
	if err != nil {
		log.Fatal(err)
	}
	performed, err := api.PerformOperation(ctx, resource.ID("NODE_ID"), "reboot",
		request.Field("type", "SOFT"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(checked.ActionID, recovered.ActionID, performed.ActionID)
}
