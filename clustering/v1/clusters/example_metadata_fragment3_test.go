package clusters_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_clustersDeleteMetadata() {
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
	result, err := service.Clusters.DeleteMetadata(ctx, resource.ID("CLUSTER_ID"),
		[]string{"team", "nullable"})
	if err != nil {
		log.Fatal(err)
	}
	result, err = service.Clusters.DeleteMetadata(ctx, resource.ID("CLUSTER_ID"), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Changed)
}
