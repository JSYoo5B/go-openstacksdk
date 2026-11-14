package profiles_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
)

func Example_profilesFragment1() {
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
	spec := map[string]any{
		"type":       "os.nova.server",
		"version":    "1.0",
		"properties": map[string]any{"name": "worker", "flavor": "FLAVOR_ID", "image": "IMAGE_ID"},
	}
	profile, err := service.Profiles.Create(ctx, profiles.CreateOpts{Name: "worker_template"},
		profiles.WithCreateSpec(spec),
		profiles.WithCreateMetadata(map[string]any{"role": "worker"}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(profile.ID, profile.UserMetadata, profile.Header)
}
