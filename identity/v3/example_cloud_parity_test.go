package v3_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
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

func Example_createProject() {
	ctx := context.Background()
	svc := identityV3(ctx)
	enabled := true
	p, err := svc.Projects.Create(ctx, projects.CreateOpts{
		Name: "demo", DomainID: "default", Enabled: &enabled,
		Extra: map[string]any{"description": nil},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(p.ID)
}

func Example_deleteService() {
	ctx := context.Background()
	svc := identityV3(ctx)
	found, err := svc.Services.Find(ctx, resource.ID("nova"), resource.WithIgnoreMissing())
	if err == nil && found == nil {
		found, err = svc.Services.Find(ctx, resource.Name("nova"), resource.WithIgnoreMissing())
	}
	if err == nil && found != nil {
		err = svc.Services.Remove(ctx, resource.ID(found.ID))
	}
	if err != nil {
		log.Fatal(err)
	}
}

func Example_deleteUser() {
	ctx := context.Background()
	svc := identityV3(ctx)
	user, err := svc.Users.FindIdentity(ctx, "alice", resource.WithIdentityFindQuery("domain_id", "default"))
	if err == nil && user != nil {
		err = svc.Users.Remove(ctx, resource.ID(user.ID))
	}
	if err != nil {
		log.Fatal(err)
	}
}

func Example_createEndpoint() {
	ctx := context.Background()
	disabled := false
	ep, err := endpoints.New(identityV3(ctx).RawClient()).Create(ctx,
		endpoints.CreateOpts{Availability: gophercloud.AvailabilityPublic, URL: "https://nova", ServiceID: "svc-1", Enabled: &disabled},
		request.Field("region_id", "RegionOne"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ep.ID)
}
