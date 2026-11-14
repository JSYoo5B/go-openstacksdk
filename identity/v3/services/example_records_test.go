package services_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printValueRecords(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", encoded)
}
func Example_records() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	svc, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	identifier := os.Getenv("IDENTITY_SERVICE_ID")
	if identifier == "" {
		log.Fatal("set IDENTITY_SERVICE_ID to a readable service ID or name")
	}
	rows, err := svc.Services.AllRecords(ctx,
		request.Query("type", "compute"),
		request.Query("is_enabled", true),
		request.MaxItems(5),
		request.Custom(func(c *services.RecordListOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		printValueRecords(row.Resource)
	}
	proxy, err := svc.Services.ProxyListRecords(ctx, services.WithRecordListJMESPath("[*].id"))
	if err != nil {
		log.Fatal(err)
	}
	printValueRecords(proxy.JSON)
	cloud := identity.NewServiceReadCloud(svc.Services)
	found, err := cloud.SearchServices(ctx, "n*",
		identity.WithServiceReadFilters(json.RawMessage(`"length(@)"`)),
		request.Custom(func(c *identity.ServiceReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	printValueRecords(found.Value)
	direct, err := cloud.GetService(ctx, identifier, request.Custom(func(c *identity.ServiceReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	printValueRecords(direct.Value)
	searched, err := cloud.GetService(ctx, identifier,
		identity.WithServiceReadFilters(json.RawMessage(`{}`)),
		request.Custom(func(c *identity.ServiceReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	printValueRecords(searched.Value)
}
