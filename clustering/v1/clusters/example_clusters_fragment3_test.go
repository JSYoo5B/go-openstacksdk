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
	api := clusters.New(client)
	for cluster, err := range api.List(ctx,
		clusters.WithListOptions(clusters.ListOpts{Status: "ACTIVE", Limit: 20}),
		request.Custom(func(c *request.Config[clusters.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
		clusters.WithListFilter("metadata", map[string]any{"team": "infra"}),
	) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(cluster.ID, cluster.Name)
	}
	cluster, err := api.Find(ctx, resource.Name("workers"))
	if err != nil {
		log.Fatal(err)
	}
	if cluster != nil {
		fmt.Println(cluster.ID)
	}
}
