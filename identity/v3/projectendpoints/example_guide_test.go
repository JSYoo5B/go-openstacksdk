package projectendpoints_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints"
	"github.com/JSYoo5B/go-openstacksdk/request"
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

func ExampleAPI_AllRecords() {
	ctx := context.Background()
	values, err := projectendpoints.New(identityV3(ctx).RawClient()).AllRecords(ctx, "project-id",
		request.Query("interface", "public"),
		request.Query("is_enabled", true))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(values))
}
