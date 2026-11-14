package v1_test

import (
	"context"
	"fmt"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_waitingFragment1() {
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
	cluster, err := service.Clusters.WaitForStatus(ctx,
		resource.ID("CLUSTER_ID"), "ACTIVE",
		resource.WithTimeout(3*time.Minute))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cluster.ID, cluster.Status, cluster.Header, cluster.Body)
}
