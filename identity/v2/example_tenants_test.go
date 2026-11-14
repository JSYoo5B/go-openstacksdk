package v2_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/identity/v2"
	"github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func identityV2(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example_findTenant() {
	ctx := context.Background()
	api := tenants.New(identityV2(ctx).RawClient())
	nameOrID := "demo"
	tenant, err := api.Find(ctx, resource.ID(nameOrID), resource.WithIgnoreMissing())
	if err == nil && tenant == nil {
		tenant, err = api.Find(ctx, resource.Name(nameOrID), resource.WithIgnoreMissing())
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(tenant)
}
