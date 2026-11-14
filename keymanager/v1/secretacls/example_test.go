package secretacls_test

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/keymanager/v1"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretacls"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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

func Example() {
	ctx := context.Background()
	scope, err := keyManager(ctx).SecretACLs.InSecret(ctx, resource.ID("secret-id"))
	if err != nil {
		log.Fatal(err)
	}
	acl, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(acl.Read))
	if _, err := scope.Set(ctx, secretacls.ACLInput{Read: json.RawMessage(`{"users":["user-id"],"project-access":false}`)}); err != nil {
		log.Fatal(err)
	}
	if _, err := scope.Delete(ctx); err != nil {
		log.Fatal(err)
	}
}
