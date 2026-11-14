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

func Example_clustersFragment6() {
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
	added, err := api.AddNodes(ctx, resource.ID("CLUSTER_ID"), clusters.AddNodesOpts{
		Nodes: []string{"NODE_ID"},
	})
	if err != nil {
		log.Fatal(err)
	}
	removed, err := api.RemoveNodes(ctx, resource.ID("CLUSTER_ID"), clusters.RemoveNodesOpts{
		Nodes: []string{"NODE_ID"},
	}, request.Custom(func(c *request.Config[clusters.RemoveNodesOpts]) error {
		c.Options.DestroyAfterDeletion = request.Present(false)
		return nil
	}))
	if err != nil {
		log.Fatal(err)
	}
	replaced, err := api.ReplaceNodes(ctx, resource.ID("CLUSTER_ID"), clusters.ReplaceNodesOpts{
		Nodes: map[string]string{"OLD_NODE": "NEW_NODE"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(added.ActionID, removed.ActionID, replaced.ActionID)
}
