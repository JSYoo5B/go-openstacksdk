package clusterpolicies_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_inCluster() {
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
	listingAPI := clusterpolicies.New(client)
	listingScope, err := listingAPI.InCluster(ctx, resource.ID("CLUSTER_ID"))
	if err != nil {
		log.Fatal(err)
	}
	values, err := listingScope.All(ctx, request.MaxItems(50))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(values))
	for value, err := range listingScope.List(ctx, request.Paginated(false)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(value.PolicyID)
	}
}
