package v3_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/limits"
)

func Example_parityCatalogFragment3() {
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
	rows, err := limits.New(client).BatchCreate(ctx, limits.BatchCreateOpts{
		{ServiceID: "svc-1", ProjectID: "p", ResourceName: "cores", ResourceLimit: 10},
	})
	limit := rows[0]

	_ = limit
}
