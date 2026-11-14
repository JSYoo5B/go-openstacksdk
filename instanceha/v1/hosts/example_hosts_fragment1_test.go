package hosts_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/instanceha/v1"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/hosts"
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
	scope, err := hosts.New(client).InSegment(ctx, resource.Name("compute-a"))
	if err != nil {
		log.Fatal(err)
	}
	host, err := scope.Create(ctx, hosts.CreateOpts{
		Name: "compute-01", Type: "COMPUTE", ControlAttributes: "SSH",
	})
	if err != nil {
		log.Fatal(err)
	}
	maintenance := true
	host, err = scope.Update(ctx, resource.ID(host.UUID),
		hosts.UpdateOpts{OnMaintenance: &maintenance})
	if err != nil {
		log.Fatal(err)
	}
	err = scope.Delete(ctx, resource.ID(host.UUID))
}
