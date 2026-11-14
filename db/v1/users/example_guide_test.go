package users_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/db/v1"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"os"
)

func database(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	password := os.Getenv("DB_PASSWORD")
	scope, err := database(ctx).Users.InInstance(ctx, resource.Name("db-server"))
	if err != nil {
		log.Fatal(err)
	}
	user, err := scope.Find(ctx, resource.Name("reader"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(user.Name, user.Host)
	if err := scope.Create(ctx, users.CreateOpts{Name: "writer", Password: password, Databases: databases.BatchCreateOpts{{Name: "app"}}}); err != nil {
		log.Fatal(err)
	}
	if err := scope.CreateBatch(ctx, users.BatchCreateOpts{
		{Name: "reader", Password: password, Host: "%"},
		{Name: "reader", Password: password, Host: "192.0.2.10"},
	}); err != nil {
		log.Fatal(err)
	}
}
