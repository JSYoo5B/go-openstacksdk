package v3_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_parityUsersFragment1() {
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
	client := service.RawClient()
	enabled := false
	project, err := projects.New(client).Create(ctx, projects.CreateOpts{Name: "demo", DomainID: "default", Enabled: &enabled})
	empty := []string{}
	project, err = projects.New(client).Update(ctx, "p-1",
		projects.UpdateOpts{Tags: &empty, Extra: map[string]any{"description": nil}},
		request.Field("id", "p-1"))

	_ = project
}
