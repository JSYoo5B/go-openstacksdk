package v2_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/identity/v2"
	"github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_parityFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.IdentityV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	enabled := true
	tenant, err := tenants.New(client).Create(ctx, tenants.CreateOpts{Name: "demo", Description: "d", Enabled: &enabled})
	tenant, err = tenants.New(client).Update(ctx, "t-1", tenants.UpdateOpts{Name: "renamed"}, request.Field("id", "t-1"))

	_ = tenant
}
