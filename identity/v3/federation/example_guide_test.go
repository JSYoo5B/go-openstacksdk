package federation_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
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

func ExampleAPI_GetMappingRecord() {
	ctx := context.Background()
	m, err := federation.New(identityV3(ctx).RawClient()).GetMappingRecord(ctx, "corp-mapping")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(*m.SchemaVersion, string(m.Rules))
}
