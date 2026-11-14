package databases_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/db/v1"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_databasesInInstance() {
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
	scope, err := service.Databases.InInstance(ctx, resource.Name("db-server"))
	if err != nil {
		log.Fatal(err)
	}

	database, err := scope.Find(ctx, resource.Name("app"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(database.Name, database.CharSet, database.Collate)

	if err := scope.Create(ctx, databases.CreateOpts{
		Name:    "logs",
		CharSet: "utf8mb4",
		Collate: "utf8mb4_unicode_ci",
	}); err != nil {
		log.Fatal(err)
	}
}
