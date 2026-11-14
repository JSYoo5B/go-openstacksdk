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

func Example_update() {
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
	node, err := api.Update(ctx, resource.Name("worker_1"), nodes.UpdateOpts{Role: request.Null[string]()},
		request.Custom(func(c *request.Config[nodes.UpdateOpts]) error {
			c.Options.Name = request.Present("worker_2")
			return nil
		}),
		request.Custom(func(c *request.Config[nodes.UpdateOpts]) error {
			c.Options.Tainted = request.Present(false)
			return nil
		}), nodes.WithUpdateMetadata(map[string]any{}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(node.ID, node.Operation.ActionID)
}
