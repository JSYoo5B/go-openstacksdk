package drivers_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_driversFragment1() {
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
	driver, err := service.Drivers.Find(ctx, resource.ID("redfish"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(driver.Name, driver.Hosts)
}
