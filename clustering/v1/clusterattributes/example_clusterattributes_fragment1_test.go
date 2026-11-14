package clusterattributes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_clusterattributesFragment1() {
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
	scope, err := service.ClusterAttributes.InCluster(ctx,
		resource.ID("CLUSTER_NAME_OR_ID"), "$.status")
	if err != nil {
		log.Fatal(err)
	}
	for attr, err := range scope.List(ctx) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(attr.NodeID, string(attr.Value))
	}
}
