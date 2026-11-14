package clusters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_clustersFragment5() {
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
	scaled, err := api.ScaleOut(ctx, resource.ID("CLUSTER_ID"), clusters.ScaleOutOpts{},
		request.Custom(func(c *request.Config[clusters.ScaleOutOpts]) error { c.Options.Count = request.Present(1); return nil }))
	if err != nil {
		log.Fatal(err)
	}
	resized, err := api.Resize(ctx, resource.ID("CLUSTER_ID"), clusters.ResizeOpts{},
		request.Custom(func(c *request.Config[clusters.ResizeOpts]) error {
			c.Options.AdjustmentType = request.Present(clusters.ExactCapacity)
			return nil
		}),
		request.Custom(func(c *request.Config[clusters.ResizeOpts]) error {
			c.Options.Number = request.Present(json.Number("3"))
			return nil
		}), request.Custom(func(c *request.Config[clusters.ResizeOpts]) error { c.Options.MinSize = request.Present(0); return nil }),
		request.Custom(func(c *request.Config[clusters.ResizeOpts]) error {
			c.Options.Strict = request.Present(false)
			return nil
		}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(scaled.ActionID, resized.ActionID)
}
