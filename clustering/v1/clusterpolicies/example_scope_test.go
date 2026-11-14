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
)

func clusteringScope(ctx context.Context) *v1.Service {
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

func Example_scopeGet() {
	ctx := context.Background()
	api := clusterpolicies.New(clusteringScope(ctx).RawClient())
	scope, err := api.InCluster(ctx, resource.ID("CLUSTER_NAME_OR_ID"))
	if err != nil {
		log.Fatal(err)
	}
	binding, err := scope.Get(ctx, "POLICY_NAME_OR_ID")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(binding.ID, binding.PolicyID, binding.ClusterID, binding.URIClusterID)
	for binding, err := range scope.List(ctx, request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error {
		v := false
		c.Options.Enabled = &v
		return nil
	})) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(binding.PolicyName, binding.IsEnabled)
	}
}
