package roles_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
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

func ExampleAPI_AllAssignmentRecords() {
	ctx := context.Background()
	values, err := roles.New(identityV3(ctx).RawClient()).AllAssignmentRecords(ctx,
		request.Query("user_id", "user-id"),
		request.Query("scope", map[string]any{"OS-INHERIT:inherited_to": "projects"}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(values))
}
