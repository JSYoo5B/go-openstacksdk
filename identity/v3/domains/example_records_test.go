package domains_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONRecords(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
}

func Example_records() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for row, err := range service.Domains.ListRecords(ctx,
		domains.WithRecordListQuery("is_enabled", true),
		domains.WithRecordListQuery("description", "operations"),
		request.Limit(100),
		request.MaxItems(100),
	) {
		if err != nil {
			log.Fatal(err)
		}
		printJSONRecords(row.Resource.Body)
	}

	cloud := identity.NewDomainReadCloud(service.Domains)
	listed, err := cloud.ListDomains(ctx,
		identity.WithDomainReadFilters(json.RawMessage(`{"is_enabled":true,"description":"operations"}`)),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords(listed.JSON)

	serverFiltered, err := cloud.SearchDomains(ctx,
		identity.WithDomainReadFilters(json.RawMessage(`{"name":"team"}`)),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords(serverFiltered.Value)

	locallyFiltered, err := cloud.SearchDomains(ctx,
		request.Custom(func(c *identity.DomainReadOpts) error { v := "team-*"; c.NameOrID = &v; return nil }),
		identity.WithDomainReadFilters(json.RawMessage(`{"description":"operations"}`)),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecords(locallyFiltered.Value)
}
