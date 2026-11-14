package v1_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clustertemplates"
)

func Example_deleteCOECluster() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ContainerInfraV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	deleted, err := service.DeleteCOECluster(ctx, "my-cluster") // false, nil when nothing matches
	id, err := service.UpdateCOECluster(ctx, "my-cluster", []clusters.UpdateOpts{
		{Op: clusters.ReplaceOp, Path: "/node_count", Value: 3},
	})
	deleted, err = service.DeleteClusterTemplate(ctx, "my-template")
	template, err := service.UpdateClusterTemplate(ctx, "my-template", []clustertemplates.UpdateOpts{
		{Op: clustertemplates.ReplaceOp, Path: "/public", Value: false},
	})

	_ = deleted
	_ = id
	_ = template
}
