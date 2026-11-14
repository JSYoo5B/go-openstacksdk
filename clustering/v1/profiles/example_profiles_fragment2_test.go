package profiles_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
)

func Example_profilesFragment2() {
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
	checked, err := service.Profiles.Validate(ctx, profiles.ValidateOpts{},
		profiles.WithValidateSpec(map[string]any{
			"type": "os.nova.server", "version": "1.0",
			"properties": map[string]any{"flavor": "FLAVOR_ID", "image": "IMAGE_ID"},
		}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(checked.Spec, checked.Body["id"])
}
