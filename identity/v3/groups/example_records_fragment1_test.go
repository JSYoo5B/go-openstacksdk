package groups_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
)

func Example_recordsFragment1() {
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
	proxy, err := service.Groups.ProxyListRecords(ctx,
		groups.WithRecordListJMESPath("[*].id"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(proxy.JSON))
}
