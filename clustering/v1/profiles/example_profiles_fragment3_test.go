package profiles_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_profilesFragment3() {
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
	service, err = conn.Clustering(ctx)
	if err != nil {
		log.Fatal(err)
	}
	updated, err := service.Profiles.Update(ctx, resource.Name("worker_template"), profiles.UpdateOpts{},
		profiles.WithUpdateMetadata(map[string]any{}),
		request.Header("X-Vendor-Mode", "explicit"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated.ID, updated.UserMetadata)
}
