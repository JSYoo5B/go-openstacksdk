package profiletypes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
)

func Example_get() {
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
	client := service.RawClient()
	api := profiletypes.New(client)
	profile, err := api.Get(ctx, "os.nova.server-1.0")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(profile.Name, profile.Schema)

	for profile, err := range api.List(ctx) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(profile.Name, profile.Version)
	}

	// Configure the selected version before concurrent client use.
	client.Microversion = "1.4"
	ops, err := api.Operations(ctx, "os.nova.server-1.0")

	_ = ops
}
