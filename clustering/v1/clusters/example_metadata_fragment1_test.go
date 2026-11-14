package clusters_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_metadataFragment1() {
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
	view, err := service.Clusters.FetchMetadata(ctx, resource.ID("CLUSTER_NAME_OR_ID"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(view.Present, view.Null, string(view.Values["team"]))
}
