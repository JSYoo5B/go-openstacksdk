package network_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_networksFind() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	network, err := service.Networks.Find(ctx, resource.Name("private"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(network.ID)

	for network, err := range service.Networks.List(ctx,
		resource.WithQuery("router:external", "true"), resource.WithPageSize(100)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(network.ID)
	}
}
