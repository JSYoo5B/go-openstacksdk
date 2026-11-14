package clusters_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_clustersFragment2() {
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
	api := clusters.New(client)
	cluster, err := api.Update(ctx, resource.Name("workers"), clusters.UpdateOpts{Timeout: request.Null[int]()},
		request.Custom(func(c *request.Config[clusters.UpdateOpts]) error {
			c.Options.Name = request.Present("workers_v2")
			return nil
		}),
		clusters.WithUpdateMetadata(map[string]any{}), clusters.WithUpdateProfileOnly(false))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cluster.ID, cluster.Operation.ActionID)
}
