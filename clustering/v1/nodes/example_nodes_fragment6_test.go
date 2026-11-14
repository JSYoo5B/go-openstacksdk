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

func Example_nodesFragment6() {
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
	node, err := api.Adopt(ctx, nodes.AdoptOpts{
		Identity: "PHYSICAL_RESOURCE_ID", Type: "os.nova.server-1.0",
	}, request.Custom(func(c *request.Config[nodes.AdoptOpts]) error {
		c.Options.Name = request.Present("adopted_worker")
		return nil
	}), request.Custom(func(c *request.Config[nodes.AdoptOpts]) error {
		c.Options.Snapshot = request.Present(false)
		return nil
	}),
		nodes.WithAdoptMetadata(map[string]any{}))
	if err != nil {
		log.Fatal(err)
	}
	preview, err := api.AdoptPreview(ctx, nodes.AdoptPreviewOpts{
		Identity: "PHYSICAL_RESOURCE_ID", Type: "os.nova.server-1.0",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(node.ID, node.PhysicalID, preview.Spec.Type, string(preview.Spec.Version))
}
