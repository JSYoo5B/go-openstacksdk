package v1_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
)

func Example_getMachine() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.BareMetalV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	record, err := service.GetMachine(ctx, "node-a")
	if err != nil || record == nil {
		log.Fatal(err)
	}
	if err := service.SetMachinePowerOn(ctx, "node-a"); err != nil {
		log.Fatal(err)
	}
	if err := conn.AttachPortToMachine(ctx, "node-a", "port-name"); err != nil {
		log.Fatal(err)
	}
}
