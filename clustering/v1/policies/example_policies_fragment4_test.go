package policies_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_policiesLoad() {
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
	service, err = conn.Clustering(ctx)
	if err != nil {
		log.Fatal(err)
	}
	tracked, err := service.Policies.Load(ctx, resource.ID("POLICY_ID"))
	if err != nil {
		log.Fatal(err)
	}
	if err := tracked.Edit(policies.UpdateOpts{}, request.Custom(func(c *request.Config[policies.UpdateOpts]) error {
		v := "scale_out_workers"
		c.Options.Name = &v
		return nil
	})); err != nil {
		log.Fatal(err)
	}
	policy, err := tracked.Commit(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(policy.Name, tracked.Dirty(), tracked.Response().Body)
}
