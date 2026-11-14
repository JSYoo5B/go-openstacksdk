package clusters_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_clustersCreate() {
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
	cluster, err := service.Clusters.Create(ctx, clusters.CreateOpts{
		Name: "workers", ProfileID: "PROFILE_ID", Timeout: request.Null[int](),
	}, request.Custom(func(c *request.Config[clusters.CreateOpts]) error { v := 0; c.Options.MinSize = &v; return nil }), request.Custom(func(c *request.Config[clusters.CreateOpts]) error { v := -1; c.Options.MaxSize = &v; return nil }),
		request.Custom(func(c *request.Config[clusters.CreateOpts]) error { v := 1; c.Options.DesiredCapacity = &v; return nil }),
		clusters.WithCreateMetadata(map[string]any{"team": "infra"}))
	if err != nil {
		log.Fatal(err)
	}
	cluster, err = service.Clusters.Get(ctx, cluster.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cluster.ID, cluster.Status, cluster.NodeIDs)
}
