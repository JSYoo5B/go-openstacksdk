package segments_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/instanceha/v1"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_create() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.InstanceHAV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	api := segments.New(client) // catalog service type "instance-ha", v1 endpoint
	segment, err := api.Create(ctx, segments.CreateOpts{
		Name: "compute-a", RecoveryMethod: "auto", ServiceType: "COMPUTE",
	})
	if err != nil {
		log.Fatal(err)
	}
	name := "compute-primary"
	segment, err = api.Update(ctx, resource.ID(segment.UUID),
		segments.UpdateOpts{Name: &name})
	if err != nil {
		log.Fatal(err)
	}
	err = api.Delete(ctx, resource.ID(segment.UUID)) // missing is ignored
}
