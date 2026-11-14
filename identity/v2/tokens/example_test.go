package tokens_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/identity/v2"
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

func ExampleAPI_Get() {
	ctx := context.Background()
	auth, err := identityV2(ctx).Tokens.Get(ctx, "token-id")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(auth.Token.ID, auth.Token.ExpiresAt, auth.User.Name)
	for _, entry := range auth.Catalog.Entries {
		fmt.Println(entry.Type, entry.Endpoints)
	}
	fmt.Println(auth.Header.Get("X-Openstack-Request-Id"))
}
