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

func Example_clustersFragment7() {
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
	checked, err := api.Check(ctx, resource.ID("CLUSTER_ID"))
	if err != nil {
		log.Fatal(err)
	}
	recovered, err := api.Recover(ctx, resource.ID("CLUSTER_ID"), clusters.RecoverOpts{},
		request.Custom(func(c *request.Config[clusters.RecoverOpts]) error {
			c.Options.Check = request.Present(false)
			return nil
		}), request.Custom(func(c *request.Config[clusters.RecoverOpts]) error {
			c.Options.CheckCapacity = request.Present(false)
			return nil
		}))
	if err != nil {
		log.Fatal(err)
	}
	operated, err := api.PerformOperation(ctx, resource.ID("CLUSTER_ID"), "reboot",
		clusters.PerformOperationOpts{},
		clusters.WithPerformOperationFilters(map[string]string{"role": "worker"}),
		clusters.WithPerformOperationParams(map[string]string{"type": "SOFT"}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(checked.ActionID, recovered.ActionID, operated.ActionID)
}
