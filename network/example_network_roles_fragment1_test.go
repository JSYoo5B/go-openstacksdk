package network_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/network"
)

func Example_networkRolesFragment1() {
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
	roles, err := service.Roles.Discover(ctx)
	if err != nil {
		log.Fatal(err)
	}
	_ = roles
	service.Roles.Reset()
}
