package v1_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1"
)

func Example_listMagnumServices() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ContainerInfraV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for item, err := range service.ListMagnumServices(ctx) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.Binary, item.Host, item.State)
	}
}
