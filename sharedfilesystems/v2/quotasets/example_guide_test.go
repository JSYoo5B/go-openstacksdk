package quotasets_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	v2 "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotasets"
)

func sharedFileSystems(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.SharedFileSystemV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	scope, err := quotasets.New(sharedFileSystems(ctx).RawClient()).InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	quota, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(quota.Shares)
	zero := int64(0)
	if _, err := scope.Update(ctx, quotasets.UpdateOpts{Shares: &zero},
		request.Custom(func(c *request.Config[quotasets.UpdateOpts]) error { v := false; c.Options.Force = &v; return nil })); err != nil {
		log.Fatal(err)
	}
}

func Example_scoped() {
	ctx := context.Background()
	parent, err := quotasets.New(sharedFileSystems(ctx).RawClient()).InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	user, err := parent.InUser(ctx, resource.ID("user-id"))
	if err != nil {
		log.Fatal(err)
	}
	count := int64(10)
	if _, err := user.Update(ctx, quotasets.UpdateOpts{Shares: &count}); err != nil {
		log.Fatal(err)
	}
	shareType, err := parent.InShareType(ctx, resource.Name("gold"))
	if err != nil {
		log.Fatal(err)
	}
	quota, err := shareType.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(quota.ShareTypeID)
}
