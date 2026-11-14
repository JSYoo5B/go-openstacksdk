package projects_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONRecordReads(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_recordReads() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	domain := os.Getenv("OS_PROJECT_DOMAIN_ID")

	for project, err := range service.Projects.ListRecords(ctx,
		request.Query("domain_id", domain),
		request.Limit(2), request.MaxItems(2)) {
		if err != nil {
			log.Fatal(err)
		}
		printJSONRecordReads(project.Resource)
	}
	project, err := service.Projects.FindRecord(ctx, "demo",
		request.Query("domain_id", domain),
		request.Query("skip_cache", true),
		request.Query("error_message", "Project lookup failed"))
	if err != nil {
		log.Fatal(err)
	}
	if project != nil {
		printJSONRecordReads(project.Resource)
	}

	filters := identity.WithProjectReadFilters(json.RawMessage(`{"is_enabled":true}`))
	listed, err := conn.ListProjects(ctx,
		request.Custom(func(c *identity.ProjectReadOpts) error { c.DomainID = domain; return nil }), filters)
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecordReads(listed.JSON)
	searched, err := conn.SearchProjects(ctx, "demo*",
		request.Custom(func(c *identity.ProjectReadOpts) error { c.DomainID = domain; return nil }), filters)
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecordReads(searched.Value)
}
