package clusterpolicies_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"time"
)

func clustering(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	scope, err := clustering(ctx).ClusterPolicies.InCluster(ctx, resource.Name("workers"))
	if err != nil {
		log.Fatal(err)
	}
	bindings, err := scope.All(ctx,
		request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error {
			v := false
			c.Options.Enabled = &v
			return nil
		}),
		request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error {
			c.Options.PolicyName = "POLICY_NAME"
			return nil
		}),
		request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error {
			c.Options.PolicyType = "senlin.policy.scaling-1.0"
			return nil
		}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(bindings))
	if err := scope.WaitForDelete(ctx, resource.ID("POLICY_NAME_OR_ID"), resource.WithTimeout(30*time.Second), resource.WithPollInterval(time.Second)); err != nil {
		log.Fatal(err)
	}
}
