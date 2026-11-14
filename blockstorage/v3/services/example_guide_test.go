package services_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func blockStorageV3(ctx context.Context) *v3.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_GetLogLevels() {
	ctx := context.Background()
	api := services.New(blockStorageV3(ctx).RawClient())
	for svc, err := range api.List(ctx, services.WithListOptions(services.ListOpts{Binary: "cinder-volume"})) {
		if err != nil {
			log.Fatal(err)
		}
		if _, err := api.Disable(ctx, services.TargetOf(svc), request.Custom(func(c *services.ActionOpts) error { c.Reason = "maintenance"; return nil })); err != nil {
			log.Fatal(err)
		}
		break
	}
	levels, err := api.GetLogLevels(ctx, request.Custom(func(c *services.LogOpts) error { v := "cinder-api"; c.Binary = &v; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	for _, level := range levels.Levels {
		fmt.Println(level.Host, level.Levels)
	}
}
