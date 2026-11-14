package compute_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_computeFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	server, err := service.Servers.Find(ctx, resource.Name("web-01"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(server.ID)

	for server, err := range service.Servers.List(ctx, resource.WithStatus("ACTIVE")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(server.ID, server.Name)
	}
}
