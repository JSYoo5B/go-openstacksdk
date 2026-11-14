package conductors_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx, openstack.WithMicroversion(openstack.BareMetal, "1.49"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BareMetalV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	conductor, err := service.Conductors.Find(ctx, resource.ID("conductor-01"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(conductor.Hostname, conductor.Alive, conductor.Drivers)
}
