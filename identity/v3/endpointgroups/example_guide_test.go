package endpointgroups_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups"
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
	api := endpointgroups.New(identityV3(ctx).RawClient())
	group, err := api.Create(ctx,
		request.Field("name", "web"),
		request.Field("filters", map[string]string{"interface": "public"}))
	if err != nil {
		log.Fatal(err)
	}
	id := group.Text("id")
	if err := api.AssociateProject(ctx, id, "project-id"); err != nil {
		log.Fatal(err)
	}
	for endpoint, err := range api.ListEndpoints(ctx, id) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(endpoint.Text("url"))
	}
	if err := api.Delete(ctx, id); err != nil {
		log.Fatal(err)
	}
}
