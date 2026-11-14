package users_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONRecords(value any) {
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
	domain := os.Getenv("USER_QUERY_DOMAIN_ID")
	if domain == "" {
		log.Fatal("set USER_QUERY_DOMAIN_ID to an allowed domain ID")
	}

	rows, err := svc.Users.AllRecords(ctx,
		users.WithRecordListQuery("domain_id", domain),
		users.WithRecordListQuery("is_enabled", true),
		request.MaxItems(10),
		request.Custom(func(c *users.RecordListOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		printJSONRecords(row.Resource.Body)
	}

	cloud := identity.NewUserReadCloud(svc.Users)
	listed, err := cloud.ListUsers(ctx,
		identity.WithUserListQuery("domain_id", domain),
		request.Custom(func(c *identity.UserListOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords(listed.Value)

	searched, err := cloud.SearchUsers(ctx, "team*",
		request.Custom(func(c *identity.UserSearchOpts) error { c.DomainID = domain; return nil }),
		identity.WithUserSearchFilters(json.RawMessage(`"length(@)"`)),
		request.Custom(func(c *identity.UserSearchOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords(searched.Value)
}
