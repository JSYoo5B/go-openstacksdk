package nodes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_nodesFragment1() {
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
	node, err := service.Nodes.Create(ctx, nodes.CreateOpts{
		Name: "worker_1", ProfileID: "PROFILE_ID",
	}, request.Custom(func(c *request.Config[nodes.CreateOpts]) error {
		c.Options.ClusterID = request.Present("CLUSTER_ID")
		return nil
	}), request.Custom(func(c *request.Config[nodes.CreateOpts]) error {
		c.Options.Role = request.Present("worker")
		return nil
	}),
		nodes.WithCreateMetadata(map[string]any{"team": "infra"}))
	if err != nil {
		log.Fatal(err)
	}
	node, err = service.Nodes.Get(ctx, node.ID, request.Custom(func(c *request.Config[nodes.GetOpts]) error { v := true; c.Options.Details = &v; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(node.ID, node.PhysicalID, node.Details)
}
