package openstack_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ExampleConnect() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	server, err := service.Servers.Find(ctx, resource.Name("example-server"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(server.ID, server.Status)
}
