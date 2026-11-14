package databases_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/db/v1"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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
	scope, err := database(ctx).Databases.InInstance(ctx, resource.Name("db-server"))
	if err != nil {
		log.Fatal(err)
	}
	if err := scope.CreateBatch(ctx, databases.BatchCreateOpts{{Name: "app"}, {Name: "logs", CharSet: "utf8mb4"}}); err != nil {
		log.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("app")); err != nil {
		log.Fatal(err)
	}
	if err := scope.WaitDeleted(ctx, resource.ID("app")); err != nil {
		log.Fatal(err)
	}
	fmt.Println("done")
}
