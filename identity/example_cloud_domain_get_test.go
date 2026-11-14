package identity_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONCloudDomainGet(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
}

func Example_cloudDomainGet() {
	id := flag.String("domain-id", "", "required domain UUID")
	name := flag.String("domain-name", "", "optional domain name")
	flag.Parse()
	if *id == "" {
		log.Fatal("-domain-id is required")
	}
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	row, err := service.Domains.GetRecord(ctx, *id)
	if err != nil {
		log.Fatal(err)
	}
	printJSONCloudDomainGet(row.Resource.Body)

	cloud := identity.NewDomainReadCloud(service.Domains)
	result, err := cloud.GetDomain(ctx,
		request.Custom(func(c *identity.DomainGetOpts) error { v := *id; c.DomainID = &v; return nil }),
		identity.WithDomainGetFilters(json.RawMessage(`{invalid-but-ignored`)),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSONCloudDomainGet(result.Record.Resource.Body)
	fmt.Println("deprecated filters ignored:", result.DeprecatedFilters)
	if *name != "" {
		found, err := cloud.GetDomain(ctx, request.Custom(func(c *identity.DomainGetOpts) error { v := *name; c.NameOrID = &v; return nil }))
		if err != nil {
			log.Fatal(err)
		}
		if found.Record != nil {
			printJSONCloudDomainGet(found.Record.Resource.Body)
		}
	}
}
