package acceleratorrequests_test

import (
	"context"
	"fmt"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/accelerator/v2"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
)

func Example_acceleratorRequestsCreate() {
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
	created, err := accel.AcceleratorRequests.Create(ctx,
		acceleratorrequests.CreateOpts{DeviceProfileName: "fpga-profile"},
		request.Header("X-Custom", "example"))
	if err != nil {
		log.Fatal(err)
	}
	for _, arq := range created.Requests {
		fmt.Println(arq.UUID, arq.State, arq.DeviceProfileGroupID)
	}

}
