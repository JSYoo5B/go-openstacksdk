package inspectionrules_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/inspectionrules"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func bareMetal(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BareMetalV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	api := bareMetal(ctx).InspectionRulesAPI()
	created, err := api.Create(ctx, inspectionrules.CreateOpts{})
	if err != nil {
		log.Fatal(err)
	}
	for value, err := range api.List(ctx, request.Custom(func(c *request.Config[inspectionrules.ListOpts]) error { c.Options.Detail = true; return nil })) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(value)
	}
	deleted, err := api.Delete(ctx, created.UUID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(deleted)
}
