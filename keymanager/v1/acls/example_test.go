package acls_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/keymanager/v1"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/acls"
)

func keyManager(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_SetSecretACL() {
	ctx := context.Background()
	api := acls.New(keyManager(ctx).RawClient())
	users := []string{"user-id"}
	access := false
	ref, err := api.SetSecretACL(ctx, "secret-id", acls.SetOpts{{Type: "read", Users: &users, ProjectAccess: &access}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(*ref)
}
