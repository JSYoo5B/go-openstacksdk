package v1_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_trackingFragment3() {
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
	tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
	if err != nil {
		log.Fatal(err)
	}
	if err := tracked.RemoveMetadata(); err != nil {
		log.Fatal(err)
	}
	_, err = tracked.Commit(ctx)
	log.Fatal(err)
}
