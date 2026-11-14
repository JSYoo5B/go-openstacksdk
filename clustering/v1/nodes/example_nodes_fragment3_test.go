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

func Example_list() {
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
	for node, err := range api.List(ctx,
		nodes.WithListOptions(nodes.ListOpts{ClusterID: "CLUSTER_ID", Status: "ACTIVE", Limit: 20}),
		request.Custom(func(c *request.Config[nodes.ListOpts]) error { v := false; c.Options.ShowDetails = &v; return nil }), request.Custom(func(c *request.Config[nodes.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
		nodes.WithListFilter("metadata", map[string]any{"team": "infra"}),
	) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(node.ID, node.PhysicalID)
	}
	node, err := api.Find(ctx, resource.Name("worker_1"))
	if err != nil {
		log.Fatal(err)
	}
	if node != nil {
		fmt.Println(node.ID)
	}
}
