package domains_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONRecordMutations(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_recordMutations() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "", "create or update")
	id := flag.String("identity", "", "Domain ID for update")
	name := flag.String("name", "example-domain", "Domain create name")
	flag.Parse()
	if *mode != "create" && *mode != "update" {
		log.Fatal("mode must be create or update")
	}
	if *mode == "update" && *id == "" {
		log.Fatal("identity is required for update")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []domains.RecordMutationOption{
		request.Field("description", nil),
		request.Field("is_enabled", false),
		request.Field("options", map[string]any{"immutable": false}),
	}
	var row *domains.DomainRecord
	if *mode == "create" {
		options = append(options, request.Field("name", *name))
		row, err = service.Domains.CreateRecord(ctx, options...)
	} else {
		row, err = service.Domains.UpdateRecord(ctx, *id, options...)
	}
	if err != nil {
		log.Fatal(err)
	}
	printJSONRecordMutations(row.Resource)
	if *mode == "update" {
		unchanged, err := service.Domains.UpdateRecord(ctx, *id)
		if err != nil {
			log.Fatal(err)
		}
		printJSONRecordMutations(unchanged.Resource)
	}
}
