package federationprotocols_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols"
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

func ExampleAPI_Create() {
	ctx := context.Background()
	api := federationprotocols.New(identityV3(ctx).RawClient())
	if _, err := api.Create(ctx, "corp", "saml2", request.Field("mapping_id", "corp-mapping")); err != nil {
		log.Fatal(err)
	}
	protocol, err := api.Find(ctx, "corp", "saml2")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(protocol.ID)
	if err := api.Delete(ctx, "corp", protocol.ID); err != nil {
		log.Fatal(err)
	}
}
