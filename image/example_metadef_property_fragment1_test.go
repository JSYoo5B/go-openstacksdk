package image_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/image"
)

func Example_get() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.Image(ctx)
	if err != nil {
		log.Fatal(err)
	}
	scope, err := service.API.MetadefProperties.InNamespace(ctx, "OS::Compute::Libvirt")
	if err != nil {
		log.Fatal(err)
	}
	value, err := scope.Get(ctx, "hw_cpu_policy")
	if err != nil {
		log.Fatal(err)
	}

	_ = value
}
