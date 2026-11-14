package limits_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/limits"
)

func identityV3(ctx context.Context) *v3.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_ProxyListRecords() {
	ctx := context.Background()
	identity := identityV3(ctx)
	result, err := identity.Limits.ProxyListRecords(ctx, limits.WithRecordListJMESPath("[*].id"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(result.JSON))
	result, err = identity.Limits.ProxyListRecords(ctx)
	if err != nil {
		log.Fatal(err)
	}
	generator := result.Value.(*limits.RecordListGenerator)
	for row, err := range generator.Records {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(row.Resource.Body["id"]))
	}
}
