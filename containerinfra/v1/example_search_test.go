package v1_test

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1"
)

func containerInfra(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ContainerInfraV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example_search() {
	ctx := context.Background()
	service := containerInfra(ctx)
	found, err := service.SearchCOEClusters(ctx, "prod-*", v1.WithSearchFilters(json.RawMessage(`{"status":"CREATE_COMPLETE"}`)))
	if err != nil {
		log.Fatal(err)
	}
	cluster, err := service.GetCOECluster(ctx, "prod-a")
	if err != nil {
		log.Fatal(err)
	}
	template, err := service.GetClusterTemplate(ctx, "k8s")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(found, cluster, template)
}
