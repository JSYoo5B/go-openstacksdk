package roles_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONRecords(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}

func Example_records() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for role, err := range service.Roles.ListRecords(ctx,
		request.Query("description", "operations"),
		request.Limit(2), request.MaxItems(2)) {
		if err != nil {
			log.Fatal(err)
		}
		printJSONRecords(role.Resource)
	}
	cloud := identity.NewRoleReadCloud(service.Roles)
	listed, err := cloud.ListRoles(ctx,
		request.Query("description", "operations"))
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords(listed.JSON)
	selected, err := cloud.SearchRoles(ctx, "read*",
		identity.WithRoleSearchFilters(json.RawMessage(`{"options":{"immutable":false}}`)))
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords(selected.Value)
	count, err := cloud.SearchRoles(ctx, "",
		identity.WithRoleSearchFilters(json.RawMessage(`"length(@)"`)))
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords(count.Value)
}
