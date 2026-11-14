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

func Example_clustersFragment8() {
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
	attached, err := api.AttachPolicy(ctx, resource.ID("CLUSTER_ID"),
		clusters.AttachPolicyOpts{PolicyID: "POLICY_ID"})
	if err != nil {
		log.Fatal(err)
	}
	updated, err := api.UpdatePolicy(ctx, resource.ID("CLUSTER_ID"),
		clusters.UpdatePolicyOpts{PolicyID: "POLICY_ID"}, request.Custom(func(c *request.Config[clusters.UpdatePolicyOpts]) error {
			c.Options.Enabled = request.Present(false)
			return nil
		}))
	if err != nil {
		log.Fatal(err)
	}
	detached, err := api.DetachPolicy(ctx, resource.ID("CLUSTER_ID"),
		clusters.DetachPolicyOpts{PolicyID: "POLICY_ID"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(attached.ActionID, updated.ActionID, detached.ActionID)
}
