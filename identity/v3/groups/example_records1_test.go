package groups_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONRecords1(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", encoded)
}

func Example_groupsFindRecord() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}

	found, err := service.Groups.FindRecord(ctx, "ops",
		request.Query("domain_id", "domain-id"),
		request.Query("skip_cache", true),
		request.Query("error_message", "Group lookup failed"),
	)
	if err != nil {
		log.Fatal(err)
	}
	if found != nil {
		printJSONRecords1(found.Resource.Body)
	}

	for row, err := range service.Groups.ListRecords(ctx,
		groups.WithRecordListQuery("domain_id", "domain-id"),
		groups.WithRecordListQuery("description", "operations"),
		request.Limit(100),
		request.MaxItems(100),
	) {
		if err != nil {
			log.Fatal(err)
		}
		printJSONRecords1(row.Resource.Body)
	}

	cloud := identity.NewGroupReadCloud(service.Groups)
	selected, err := cloud.SearchGroups(ctx, "ops-*",
		request.Custom(func(c *identity.GroupSearchOpts) error { v := "domain-id"; c.DomainID = &v; return nil }),
		identity.WithGroupSearchFilters(json.RawMessage(`{"description":"operations"}`)),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords1(selected.Value)
}
