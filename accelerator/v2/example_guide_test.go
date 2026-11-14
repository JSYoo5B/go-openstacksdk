package accelerator_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/accelerator/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func accelerator(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Accelerator(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	accel := accelerator(ctx)
	for device, err := range accel.Devices.List(ctx, resource.WithQuery("hostname", "compute-1")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(device.UUID, device.Type, device.Status)
	}
	deployable, err := accel.Deployables.Find(ctx, resource.Name("compute-1_FPGA"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(deployable.UUID)
}
