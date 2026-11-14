package v3_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
)

func Example_assign() {
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
	api := roles.New(client)
	// Python assign_project_role_to_user("p-1", "u-1", "r-1")
	err = api.Assign(ctx, "r-1", roles.AssignOpts{ProjectID: "p-1", UserID: "u-1"})
	// Python validate_group_has_system_role("g-1", "r-1", "all")
	hasRole := api.Validate(ctx, "r-1", roles.ValidateOpts{System: true, GroupID: "g-1"}) == nil
	// Python assign_domain_role_to_group("d-1", "g-1", "r-1", inherited=True)
	err = osinherit.New(client).Assign(ctx, "r-1", osinherit.AssignOpts{DomainID: "d-1", GroupID: "g-1"})

	_ = hasRole
}
