package v1_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_nodesDelete() {
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
	submission, err := service.Nodes.Delete(ctx, resource.ID("NODE_ID"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(submission.ActionID)
	if err := service.Nodes.WaitForDelete(ctx, resource.ID("NODE_ID")); err != nil {
		log.Fatal(err)
	}
}
