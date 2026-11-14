package deployables_test

import (
	"context"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/accelerator/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"log"
)

func Example_deployablesAll() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.AcceleratorV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	_, _, _ = conn, service, client
	accel, err := conn.Accelerator(ctx)
	if err != nil {
		log.Fatal(err)
	}
	values, err := accel.Deployables.All(ctx, resource.WithQuery("name", "compute-1_FPGA"))

	_ = values
}
