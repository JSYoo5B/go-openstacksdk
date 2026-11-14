package softwareconfigs_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/softwareconfigs"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.OrchestrationV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := service.SoftwareConfigs()
	config, err := api.Create(ctx, softwareconfigs.CreateOpts{Name: request.Present("boot")},
		request.Custom(func(c *request.Config[softwareconfigs.CreateOpts]) error {
			c.Options.Group = request.Present("script")
			return nil
		}),
		softwareconfigs.WithCreateConfig("#!/bin/sh\necho hi\n"))
	if err != nil {
		log.Fatal(err)
	}
	got, err := api.Get(ctx, config.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(got.ID)
	for item, err := range api.List(ctx, request.Limit(50)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.ID)
	}
	if err := api.Delete(ctx, config.ID); err != nil {
		log.Fatal(err)
	}
}
