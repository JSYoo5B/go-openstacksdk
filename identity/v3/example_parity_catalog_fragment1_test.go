package v3_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_parityCatalogFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	// Python: conn.identity.delete_region("RegionOne")
	err = regions.New(client).Remove(ctx, resource.ID("RegionOne"))
	// Python: conn.identity.delete_region("RegionOne", ignore_missing=False)
	err = regions.New(client).Remove(ctx, resource.ID("RegionOne"), resource.WithMissingError())
}
