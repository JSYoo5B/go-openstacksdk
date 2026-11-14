package users_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/db/v1"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_usersInInstance() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.DatabaseV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	hostScope, err := service.Users.InInstance(ctx, resource.ID("instance-id"),
		users.WithHost("192.0.2.10"))
	if err != nil {
		log.Fatal(err)
	}
	id, err := hostScope.ResolveID(ctx, resource.Name("alice@office"))
	if err != nil {
		log.Fatal(err)
	}
	user, err := hostScope.Get(ctx, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(user.Name, user.Host)
	if err := hostScope.Delete(ctx, resource.ID(id)); err != nil {
		log.Fatal(err)
	}
	if err := hostScope.WaitDeleted(ctx, resource.ID(id)); err != nil {
		log.Fatal(err)
	}
}
