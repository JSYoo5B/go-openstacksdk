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

func Example_profilesFragment4() {
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
	tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
	if err != nil {
		log.Fatal(err)
	}
	if err := tracked.Edit(profiles.UpdateOpts{},
		request.Custom(func(c *request.Config[profiles.UpdateOpts]) error {
			v := "renamed_template"
			c.Options.Name = &v
			return nil
		}),
		profiles.WithUpdateMetadata(map[string]any{"role": "worker"})); err != nil {
		log.Fatal(err)
	}
	profile, err := tracked.Commit(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(profile.Name, tracked.Dirty(), tracked.Response().Body)
}
