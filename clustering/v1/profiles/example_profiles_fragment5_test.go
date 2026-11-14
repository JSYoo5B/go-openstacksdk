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

func Example_profilesFragment5() {
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
	for profile, err := range service.Profiles.List(ctx,
		profiles.WithListOptions(profiles.ListOpts{Limit: 20, Sort: "created_at:desc"}),
		request.Custom(func(c *request.Config[profiles.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
		profiles.WithListFilter("metadata", map[string]any{"role": "worker"})) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(profile.ID, profile.Name)
	}
	found, err := service.Profiles.Find(ctx, resource.Name("worker_template"))
	if err != nil {
		log.Fatal(err)
	}
	if found != nil {
		fmt.Println(found.ID)
	}
}
